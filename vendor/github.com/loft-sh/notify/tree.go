// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

package notify

const buffer = 128

type Tree interface {
	Watch(string, chan<- EventInfo, DoNotWatchFn, ...Event) error
	Stop(chan<- EventInfo)
	Close() error
}

func NewTree() Tree {
	c := make(chan EventInfo, buffer)
	w := newWatcher(c)
	if rw, ok := w.(recursiveWatcher); ok {
		return newRecursiveTree(rw, c)
	}
	return newNonrecursiveTree(w, c, make(chan EventInfo, buffer))
}
