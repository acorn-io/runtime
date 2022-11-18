package autoupgrade

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade/validate"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/pull"
	tags2 "github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/router"
	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/sirupsen/logrus"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	syncQueue       = make(chan struct{}, 1)
	ticker          *time.Ticker
	currentInterval string
	m               sync.Mutex
)

// Sync tells the daemon to trigger the image syncing logic
func Sync() {
	// This select statement lets us "rate limit" incoming syncs. Because the channel is of size one, if the receiver
	// isn't ready (because a run of the sync logic is currently in-progress) when this function is called, the default
	// case will be hit and the event will be effectively dropped.
	select {
	case syncQueue <- struct{}{}:
		logrus.Debugf("Handled a sync event")
	default:
		logrus.Debugf("Dropped a sync event")
	}
}

type daemon struct {
	client             kclient.Client
	appKeysToNextCheck map[kclient.ObjectKey]nextCheckDetails
}

// StartSync launches starts the daemon. It watches for new sync events coming and ensures a sync is triggered
// periodically.
func StartSync(ctx context.Context, client kclient.Client) error {
	cfg, err := config.Get(ctx, client)
	if err != nil {
		return err
	}

	dd, err := time.ParseDuration(*cfg.AutoUpgradeInterval)
	if err != nil {
		return fmt.Errorf("failed to parse image check interval %v: %w", cfg.AutoUpgradeInterval, err)
	}

	m.Lock()
	ticker = time.NewTicker(dd)
	currentInterval = *cfg.AutoUpgradeInterval
	m.Unlock()

	// Sync() will be called from controllers when necessary, but this also ensures it will be called periodically, in
	// case nothing has happened in a controller to trigger it. The ticker is based on cfg.AutoUpgradeInterval, which can
	// be dynamically updated by the config handler in this package.
	go func() {
		for {
			select {
			case <-ticker.C:
				Sync()
			case <-ctx.Done():
				return
			}
		}
	}()

	d := &daemon{
		client:             client,
		appKeysToNextCheck: map[kclient.ObjectKey]nextCheckDetails{},
	}

	// Trigger one sync upon startup of the daemon
	err = d.sync(ctx)
	if err != nil {
		logrus.Errorf("Encountered error syncing auto-upgrade apps: %v", err)
	}

	// Ranging over this channel allows us to receive periodic and on-demand sync events
	for {
		select {
		case <-syncQueue:
			err = d.sync(ctx)
			if err != nil {
				logrus.Errorf("Encountered error syncing auto-upgrade apps: %v", err)
			}

			//This, in combination with the select statement in Sync() limits us to a max of one run of d.sync() per second
			time.Sleep(time.Second * 1)
		case <-ctx.Done():
			logrus.Infof("Exiting auto-upgrade daemon")
			return nil
		}
	}
}

func (d *daemon) sync(ctx context.Context) error {
	logrus.Debugf("Performing auto-upgrade sync")
	cfg, err := config.Get(ctx, d.client)
	if err != nil {
		return err
	}

	defaultNextCheckInterval, err := time.ParseDuration(*cfg.AutoUpgradeInterval)
	if err != nil {
		return err
	}
	defaultNextCheck := time.Now().Add(defaultNextCheckInterval)

	// Look for any new apps that we need to add to our map
	var appInstances v1.AppInstanceList
	err = d.client.List(ctx, &appInstances)
	if err != nil {
		return err
	}

	// This loop does two things:
	// 1. Builds a general purpose map (apps) of all returned apps for use throughout the function
	// 2. Add any NEW apps with autoUpgrade turned on to the d.appKeysToNextCheck map with a next check time in the past
	//    to ensure they'll be checked this sync
	apps := map[kclient.ObjectKey]v1.AppInstance{}
	for _, app := range appInstances.Items {
		key := router.Key(app.Namespace, app.Name)
		apps[key] = app

		if _, ok := Mode(app.Spec); ok {
			if _, ok := d.appKeysToNextCheck[key]; !ok {
				// If it's not in the map yet, we should check it on this run, so set the "next check" to a time in the past
				d.appKeysToNextCheck[key] = nextCheckDetails{time: time.Now().Add(-time.Second), appSpecificInterval: ""}
			}
		}
	}

	// This loop iterates over d.appKeysToNextCheck (which represents all the apps that have autoUpgrade turned on) and does the following:
	// 1. If the app no longer exists in the general apps map, remove it, because it must no longer exist
	// 2. Checks to see if the app has a specific interval set. If it does, and that isn't the interval used on the last run, recalculate the "next check" time
	// 3. If the app no longer has autoUpgrade turned on, remove it from appKeytsToNextCheck. It must have been turned off since last run
	for k, nextCheck := range d.appKeysToNextCheck {
		app, ok := apps[k]
		if !ok {
			delete(d.appKeysToNextCheck, k)
			continue
		}

		if _, ok := Mode(app.Spec); ok {
			// Note: if we're using the default interval, nextCheck.appSpecificInterval is ""
			if nextCheck.appSpecificInterval != app.Spec.AutoUpgradeInterval {
				next, interval, err := calcNextCheck(defaultNextCheck, app)
				if err != nil {
					logrus.Errorf("Problem calculating next check time for app %v: %v", app.Name, err)
					continue
				}
				d.appKeysToNextCheck[k] = nextCheckDetails{time: next, appSpecificInterval: interval}
			}
		} else {
			// App no longer has auto-upgrade enabled. Remove it
			delete(d.appKeysToNextCheck, k)
		}
	}

	// d.appKeysToNextCheck is now fully up-to-date. This loop iterates over it and compares each app's nextCheck time
	// to the current time. If it's nextCheck is before Now, then it is time to check the app.
	// The refresh map is used to group apps by their image. Checking for new versions of an image is relatively expensive
	// because it has to go out to an external registry. So, if many apps are using the same image, we just want to pull
	// the tags for that image once.  The namespace is in the key because pull credentials are namespace specific.
	refresh := map[imageAndNamespaceKey][]kclient.ObjectKey{}
	now := time.Now()
	for appKey, nextCheck := range d.appKeysToNextCheck {
		app, ok := apps[appKey]
		if !ok {
			continue
		}

		// If next check time is before now, app is due for a check
		if nextCheck.time.Before(now) {
			img := app.Status.AppImage.Name
			if img == "" {
				img = removeTagPattern(app.Spec.Image)
			}
			imageKey := imageAndNamespaceKey{image: img, namespace: app.Namespace}
			appKeys := refresh[imageKey]
			refresh[imageKey] = append(appKeys, appKey)

		}
	}

	// This loop iterates over the refresh map and looks for new versions of image being used for each app.
	// If it determines a newer version of an image is available for an app, it will update the app with that information
	// which will trigger the appInstance handlers to pick up the change and deploy the new version of the app
	for imageKey, appsForImage := range refresh {
		current, tags, err := pull.ListTags(ctx, d.client, imageKey.namespace, imageKey.image)
		if err != nil {
			logrus.Errorf("Problem listing tags for image %v: %v", imageKey.image, err)
		}
		r, err := imagename.ParseReference(imageKey.image)
		if err != nil {
			logrus.Errorf("Problem parsing image referece %v: %v", imageKey.image, err)
		}
		localTags, err := tags2.GetTagsMatchingRepository(r, ctx, d.client, "acorn")
		if err != nil {
			logrus.Errorf("Problem finding local tags matching %v: %v", imageKey.image, err)
		}
		tags = append(tags, localTags...)

		for _, appKey := range appsForImage {
			app, ok := apps[appKey]
			if !ok {
				continue
			}

			var updated bool

			// If we have autoUpgradeTagPattern, we need to use it to compare the current tag against all the tags
			tagPattern, isPattern := AutoUpgradePattern(app.Spec.Image)
			if isPattern {
				var newTag string
				newTag, err = FindLatest(current.Identifier(), tagPattern, tags)
				if err != nil {
					logrus.Errorf("Problem finding latest tag for app %v: %v", appKey, err)
					continue
				}

				if newTag != current.Identifier() {
					updated = true
					mode, _ := Mode(app.Spec)
					t := current.Context().Tag(newTag).Name()
					// go-containerregistry adds this prefix by default. We need to strip it if it wasn't present in the original so that
					// local images work
					if strings.HasPrefix(t, "index.docker.io/library/") && !strings.HasPrefix(imageKey.image, "index.docker.io/library/") {
						t = strings.TrimPrefix(t, "index.docker.io/library/")
					}
					switch mode {
					case "enabled":
						if app.Status.AvailableAppImage == t {
							continue
						}
						app.Status.AvailableAppImage = t
						app.Status.ConfirmUpgradeAppImage = ""
					case "notify":
						if app.Status.ConfirmUpgradeAppImage == t {
							continue
						}
						app.Status.ConfirmUpgradeAppImage = t
						app.Status.AvailableAppImage = ""
					default:
						logrus.Warnf("Unrecognized auto-upgrade mode %v for %v", mode, app.Name)
						continue

					}
					logrus.Infof("Triggering an auto-upprade of app %v because a new tag was found matching pattern %v. New tag: %v",
						appKey, tagPattern, newTag)

					if err := d.client.Status().Update(ctx, &app); err != nil {
						logrus.Errorf("Problem updating %v: %v", appKey, err)
						continue
					}
				}
			}

			// Updated can be false for two reasons:
			// 1. The tag was a pattern and a newer tag was not found
			// 2. The tag was not a pattern
			// In either case, we also want to check to see if new content was pushed to the current tag
			// This satisfies the usecase of autoUpgrade with an app's tag is something static, like "latest"
			if !updated {
				digest, err := pull.ImageDigest(ctx, d.client, app.Namespace, imageKey.image)
				if err != nil {
					logrus.Errorf("Problem getting digest app %v: %v", appKey, err)
					continue
				}
				if app.Status.AppImage.Digest != digest {
					mode, _ := Mode(app.Spec)
					switch mode {
					case "enabled":
						if app.Status.AvailableAppImage == imageKey.image {
							continue
						}
						app.Status.AvailableAppImage = imageKey.image
						app.Status.ConfirmUpgradeAppImage = ""
					case "notify":
						if app.Status.ConfirmUpgradeAppImage == imageKey.image {
							continue
						}
						app.Status.ConfirmUpgradeAppImage = imageKey.image
						app.Status.AvailableAppImage = ""
					default:
						logrus.Warnf("Unrecognized auto-upgrade mode %v for %v", mode, app.Name)
						continue
					}
					logrus.Infof("Triggering an auto-upprade of app %v because a new digest was detected for image %v",
						appKey, imageKey.image)
					if err := d.client.Status().Update(ctx, &app); err != nil {
						logrus.Errorf("Problem updating %v: %v", appKey, err)
						continue
					}
				}
			}

			// This app was checked on this run, so update the nextCheck time for this app
			nextCheckTime, interval, err := calcNextCheck(defaultNextCheck, app)
			if err != nil {
				logrus.Errorf("Problem calculating next check time for app %v: %v", app.Name, err)
				continue
			}
			d.appKeysToNextCheck[appKey] = nextCheckDetails{time: nextCheckTime, appSpecificInterval: interval}
		}
	}

	nearestNextCheck := defaultNextCheck
	for _, nextCheck := range d.appKeysToNextCheck {
		if nextCheck.time.After(now) && nextCheck.time.Before(nearestNextCheck) {
			nearestNextCheck = nextCheck.time
		}
	}

	if defaultNextCheck.After(nearestNextCheck) {
		go func() {
			// Schedule the next sync for the next nearest interval
			logrus.Debugf("Next auto-upgrade sync scheduled for: %v", nearestNextCheck)
			time.Sleep(time.Until(nearestNextCheck))
			Sync()
		}()
	}

	return nil
}

func calcNextCheck(defaultNextCheck time.Time, app v1.AppInstance) (time.Time, string, error) {
	if app.Spec.AutoUpgradeInterval != "" {
		nextCheckInterval, err := time.ParseDuration(app.Spec.AutoUpgradeInterval)
		if err != nil {
			return time.Time{}, "", err
		}
		return time.Now().Add(nextCheckInterval), app.Spec.AutoUpgradeInterval, nil
	}
	return defaultNextCheck, "", nil
}

func removeTagPattern(image string) string {
	p, ok := AutoUpgradePattern(image)
	if !ok {
		return image
	}

	return strings.TrimSuffix(image, ":"+p)
}

func AutoUpgradePattern(image string) (string, bool) {
	// This first bit is adapted from https://github.com/google/go-containerregistry/blob/main/pkg/name/tag.go
	// Split on ":"
	parts := strings.Split(image, ":")
	var tag string
	// Verify that we aren't confusing a tag for a hostname w/ port for the purposes of weak validation.
	if len(parts) > 1 && !strings.Contains(parts[len(parts)-1], "/") {
		tag = parts[len(parts)-1]
	}

	return tag, strings.ContainsAny(tag, "#*")
}

func Mode(appSpec v1.AppInstanceSpec) (string, bool) {
	_, isPat := AutoUpgradePattern(appSpec.Image)
	on := appSpec.GetAutoUpgrade() || appSpec.GetNotifyUpgrade() || isPat

	if !on {
		return "", false
	}

	mode := "enabled"
	if appSpec.GetNotifyUpgrade() {
		mode = "notify"
	}

	return mode, on
}

type imageAndNamespaceKey struct {
	image     string
	namespace string
}

type nextCheckDetails struct {
	time                time.Time
	appSpecificInterval string
}

func UpdateInterval(newInterval string) error {
	m.Lock()
	defer m.Unlock()

	if ticker == nil {
		return fmt.Errorf("interval not yet initialized")
	}
	if currentInterval == newInterval {
		return nil
	}
	newDur, err := validate.AutoUpgradeInterval(newInterval)
	if err != nil {
		return err
	}

	logrus.Infof("Updating auto-upgrade sync interval to %v", newInterval)
	currentInterval = newInterval
	ticker.Reset(newDur)
	Sync()

	return nil

}
