package autoupgrade

import (
	"context"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/imageallowrules"
	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/sirupsen/logrus"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultNoReg = "xxx-no-reg"

var syncQueue = make(chan struct{}, 1)

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
	client           daemonClient
	appKeysPrevCheck map[kclient.ObjectKey]time.Time
}

func newDaemon(c kclient.Client) *daemon {
	return &daemon{
		client:           &client{client: c},
		appKeysPrevCheck: make(map[kclient.ObjectKey]time.Time),
	}
}

// StartSync starts the daemon. It watches for new sync events coming and ensures a sync is triggered
// periodically.
func StartSync(ctx context.Context, client kclient.Client) {
	d := newDaemon(client)

	// Trigger one sync upon startup of the daemon
	nextWait, err := d.sync(ctx, time.Now())
	if err != nil {
		logrus.Errorf("Encountered error syncing auto-upgrade apps: %v", err)
	}

	go func() {
		timer := time.NewTimer(nextWait)
		// Receive periodic and on-demand sync events
		for {
			select {
			case <-timer.C:
				Sync()
			case <-syncQueue:
				if !timer.Stop() {
					// Ensure the timer's channel is drained, but don't block if it is empty.
					// Reset should only be called on stopped timer whose channel is drained.
					select {
					case <-timer.C:
					default:
					}
				}
				nextWait, err = d.sync(ctx, time.Now())
				if err != nil {
					logrus.Errorf("Encountered error syncing auto-upgrade apps: %v", err)
				}

				timer.Reset(nextWait)
				// This, in combination with the select statement in Sync() limits us to a max of one run of d.sync() per second
				time.Sleep(time.Second)
			case <-ctx.Done():
				logrus.Infof("Exiting auto-upgrade daemon")
				return
			}
		}
	}()
}

func (d *daemon) sync(ctx context.Context, now time.Time) (time.Duration, error) {
	logrus.Debugf("Performing auto-upgrade sync")
	defaultNextCheckInterval, _ := time.ParseDuration(config.DefaultImageCheckIntervalDefault)
	cfg, err := d.client.getConfig(ctx)
	if err != nil {
		return defaultNextCheckInterval, err
	}

	// cfg.AutoUpgradeInterval will never be nil here because config.Get will set a default.
	if cfgNextCheckInterval, err := time.ParseDuration(*cfg.AutoUpgradeInterval); err != nil {
		logrus.Warnf("Error parsing auto-upgrade interval in config %s is invalid, using default of %s: %v", *cfg.AutoUpgradeInterval, config.DefaultImageCheckIntervalDefault, err)
	} else {
		defaultNextCheckInterval = cfgNextCheckInterval
	}

	// Look for any new apps that we need to add to our map
	appInstances, err := d.client.listAppInstances(ctx)
	if err != nil {
		return defaultNextCheckInterval, err
	}

	// This loop does two things:
	// 1. Builds a general purpose map (apps) of all returned apps for use throughout the function
	// 2. Add any NEW apps with autoUpgrade turned on to the d.appKeysPrevCheck map with a next check time in the past
	//    to ensure they'll be checked this sync
	apps := map[kclient.ObjectKey]v1.AppInstance{}
	for _, app := range appInstances {
		key := router.Key(app.Namespace, app.Name)
		apps[key] = app

		if _, ok := Mode(app.Spec); ok {
			if _, ok := d.appKeysPrevCheck[key]; !ok {
				// If it's not in the map yet, we should check it on this run, so set the "previous check" to a time in the past
				d.appKeysPrevCheck[key] = time.Time{}
			}
		}
	}

	d.refreshImages(ctx, apps, d.determineAppsToRefresh(apps, defaultNextCheckInterval, now), now)

	nearestNextCheck := now.Add(defaultNextCheckInterval)
	for appKey, prevCheck := range d.appKeysPrevCheck {
		nextCheck, err := calcNextCheck(defaultNextCheckInterval, prevCheck, apps[appKey])
		if err == nil && nextCheck.Before(nearestNextCheck) {
			nearestNextCheck = nextCheck
		}
	}

	return time.Until(nearestNextCheck), nil
}

// determineAppsToRefresh relies on the fact that d.appKeysPrevCheck is now fully up-to-date. It will iterate over it,
// calculate the next update time, and compare it to the current time. If its next update time is before Now, then it is
// time to check the app. The refresh map is used to group apps by their image. Checking for new versions of an image is
// relatively expensive because it has to go out to an external registry. So, if many apps are using the same image, we
// just want to pull the tags for that image once. The namespace is in the key because pull credentials are namespace specific.
func (d *daemon) determineAppsToRefresh(apps map[kclient.ObjectKey]v1.AppInstance, defaultNextCheckInterval time.Duration, updateTime time.Time) map[imageAndNamespaceKey][]kclient.ObjectKey {
	imagesToRefresh := map[imageAndNamespaceKey][]kclient.ObjectKey{}
	for appKey, prevCheckTime := range d.appKeysPrevCheck {
		app, appExists := apps[appKey]
		if _, ok := Mode(app.Spec); !appExists || !ok {
			// App doesn't exist or no longer has auto-upgrade enabled. Remove it
			delete(d.appKeysPrevCheck, appKey)
			continue
		}

		nextCheck, err := calcNextCheck(defaultNextCheckInterval, prevCheckTime, app)
		if err != nil {
			logrus.Errorf("Problem calculating next check time for app %v: %v", app.Name, err)
			continue
		}

		// If next check time is before now, app is due for a check
		if nextCheck.Before(updateTime) {
			img := app.Status.AppImage.Name
			if img == "" {
				img = removeTagPattern(app.Spec.Image)
			}
			imageKey := imageAndNamespaceKey{image: img, namespace: app.Namespace}
			imagesToRefresh[imageKey] = append(imagesToRefresh[imageKey], appKey)
		}
	}

	return imagesToRefresh
}

// refreshImages iterates over the imagesToRefresh map and looks for new versions of image being used for each app.
// If it determines a newer version of an image is available for an app, it will update the app with that information
// which will trigger the appInstance handlers to pick up the change and deploy the new version of the app
func (d *daemon) refreshImages(ctx context.Context, apps map[kclient.ObjectKey]v1.AppInstance, imagesToRefresh map[imageAndNamespaceKey][]kclient.ObjectKey, updateTime time.Time) {
	for imageKey, appsForImage := range imagesToRefresh {
		current, err := imagename.ParseReference(imageKey.image, imagename.WithDefaultRegistry(defaultNoReg), imagename.WithDefaultTag(""))
		if err != nil {
			logrus.Errorf("Problem parsing image referece %v: %v", imageKey.image, err)
			continue
		}

		for _, appKey := range appsForImage {
			app := apps[appKey]
			var (
				updated      bool
				newTag       string
				digest       string
				nextAppImage string
			)

			// If we have autoUpgradeTagPattern, we need to use it to compare the current tag against all the tags
			tagPattern, isPattern := AutoUpgradePattern(app.Spec.Image)
			if isPattern {
				nextAppImage, updated, err = findLatestTagForImageWithPattern(ctx, d.client, current.Identifier(), imageKey.namespace, imageKey.image, tagPattern)
				if err != nil {
					logrus.Errorf("Problem finding latest tag for app %v: %v", appKey, err)
					continue
				}
			}

			// Updated can be false for two reasons:
			// 1. The tag was a pattern and a newer tag was not found
			// 2. The tag was not a pattern
			// In either case, we also want to check to see if new content was pushed to the current tag, if the image has a current tag.
			// This satisfies the usecase of autoUpgrade with an app's tag is something static, like "latest"
			// However, if the tag is a pattern and the current image has no tag, we don't want to check for a digest because this would
			// result in a digest upgrade even though no tag matched.
			if !updated && (!isPattern || current.Identifier() != "") {
				nextAppImage = imageKey.image
				var pullErr error
				if current.Context().RegistryStr() != defaultNoReg {
					digest, pullErr = d.client.imageDigest(ctx, app.Namespace, imageKey.image)
				}

				// If we did not find the digest remotely, check to see if there is a version of this tag locally
				if digest == "" {
					if localDigest, ok, _ := d.client.resolveLocalTag(ctx, app.Namespace, imageKey.image); ok && localDigest != "" {
						digest = localDigest
					}
				}

				if digest == "" && pullErr != nil {
					logrus.Errorf("Problem getting updated digest for image %v from remote. Error: %v", imageKey.image, pullErr)
				}
			}

			if updated || strings.TrimPrefix(app.Status.AppImage.Digest, "sha256:") != strings.TrimPrefix(digest, "sha256:") {
				if !updated && digest != "" {
					if err := d.client.checkImageAllowed(ctx, app.Namespace, nextAppImage); err != nil {
						if _, ok := err.(*imageallowrules.ErrImageNotAllowed); ok {
							logrus.Debugf("Updated image %s for %s/%s is not allowed: %v", nextAppImage, app.Namespace, app.Name, err)
							d.appKeysPrevCheck[appKey] = updateTime
							continue
						}
						logrus.Errorf("error checking if updated image %s for %s/%s  is allowed: %v", app.Namespace, app.Name, nextAppImage, err)
						continue
					}
				}

				mode, _ := Mode(app.Spec)
				switch mode {
				case "enabled":
					if app.Status.AvailableAppImage == nextAppImage {
						d.appKeysPrevCheck[appKey] = updateTime
						continue
					}
					app.Status.AvailableAppImage = nextAppImage
					app.Status.ConfirmUpgradeAppImage = ""
				case "notify":
					if app.Status.ConfirmUpgradeAppImage == nextAppImage {
						d.appKeysPrevCheck[appKey] = updateTime
						continue
					}
					app.Status.ConfirmUpgradeAppImage = nextAppImage
					app.Status.AvailableAppImage = ""
				default:
					logrus.Warnf("Unrecognized auto-upgrade mode %v for %v", mode, app.Name)
					continue
				}
				if updated {
					logrus.Infof("Triggering an auto-upgrade of app %v because a new tag was found matching pattern %v. New tag: %v",
						appKey, tagPattern, newTag)
				} else {
					logrus.Infof("Triggering an auto-upgrade of app %v because a new digest [%v] was detected for image %v",
						appKey, digest, imageKey.image)
				}
				if err := d.client.updateAppStatus(ctx, &app); err != nil {
					logrus.Errorf("Problem updating %v: %v", appKey, err)
					continue
				}
			}

			// This app was checked on this run, so update the prevCheckTime time for this app
			d.appKeysPrevCheck[appKey] = updateTime
		}
	}
}

func calcNextCheck(defaultInterval time.Duration, lastUpdate time.Time, app v1.AppInstance) (time.Time, error) {
	if app.CreationTimestamp.After(lastUpdate) {
		// If the app was created after the last update time, then the app was deleted and recreated between sync runs.
		// Return a "zero" time to ensure the app gets refreshed now.
		return time.Time{}, nil
	}
	if app.Spec.AutoUpgradeInterval != "" {
		nextCheckInterval, err := time.ParseDuration(app.Spec.AutoUpgradeInterval)
		if err != nil {
			return time.Time{}, err
		}
		defaultInterval = nextCheckInterval
	}
	return lastUpdate.Add(defaultInterval), nil
}

func removeTagPattern(image string) string {
	p, ok := AutoUpgradePattern(image)
	if !ok {
		return image
	}

	return strings.TrimSuffix(image, ":"+p)
}

// AutoUpgradePattern returns the tag and a boolean indicating whether it is actually a pattern (versus a concrete tag)
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
