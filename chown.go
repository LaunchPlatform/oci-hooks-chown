package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

const (
	PolicyRecursive string = "recursive"
	PolicyRootOnly         = "root-only"
)

type ChownRequest struct {
	// The name of chown
	Name string
	// The "destination" filed of mount point to chown
	MountPoint string
	// The user (uid) to set for the mount point
	User int
	// The group (gid) to set for the mount point
	Group int
	// The policy for chown
	Policy string
}

const (
	annotationPrefix        string = "com.launchplatform.oci-hooks.mount-chown."
	annotationMountPointArg string = "mount-point"
	annotationOwnerArg      string = "owner"
	annotationPolicyArg     string = "policy"
)

func parseOwner(owner string) (int, int, error) {
	parts := strings.Split(owner, ":")
	if len(parts) < 1 || len(parts) > 2 {
		return 0, 0, fmt.Errorf("Expected only one or two parts in the owner but got %d instead", len(parts))
	}
	uid, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	if len(parts) == 1 {
		return uid, 0, nil
	}
	gid, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return uid, gid, nil
}

func parseChownRequests(annotations map[string]string) map[string]ChownRequest {
	requests := map[string]ChownRequest{}
	for key, value := range annotations {
		if !strings.HasPrefix(key, annotationPrefix) {
			continue
		}
		keySuffix := key[len(annotationPrefix):]
		parts := strings.Split(keySuffix, ".")
		name, chownArg := parts[0], parts[1]
		request, ok := requests[name]
		if !ok {
			request = ChownRequest{Name: name, User: -1, Group: -1}
		}
		if chownArg == annotationMountPointArg {
			request.MountPoint = value
		} else if chownArg == annotationOwnerArg {
			uid, gid, err := parseOwner(value)
			if err != nil {
				log.Warnf("Invalid owner argument for %s with error %s, ignored", name, err)
				continue
			}
			if uid < 0 || gid < 0 {
				log.Warnf("Invalid owner argument for %s with negative uid or gid, ignored", name)
				continue
			}
			request.User = uid
			request.Group = gid
		} else if chownArg == annotationPolicyArg {
			request.Policy = value
		} else {
			log.Warnf("Invalid chown argument %s for request %s, ignored", chownArg, name)
			continue
		}
		requests[name] = request
	}

	// Convert map from using name as the key to use mount-point instead
	mountPointRequests := map[string]ChownRequest{}
	for _, request := range requests {
		var emptyValue = false
		if request.MountPoint == "" {
			log.Warnf("Empty mount-point argument value for %s, ignored", request.Name)
			emptyValue = true
		}
		if request.User == -1 || request.Group == -1 {
			log.Warnf("Empty owner argument value for %s, ignored", request.Name)
			emptyValue = true
		}
		if request.Policy != "" && request.Policy != PolicyRecursive && request.Policy != PolicyRootOnly {
			log.Warnf("Invalid policy argument value %s for %s, ignored", request.Policy, request.Name)
			emptyValue = true
		}
		if emptyValue {
			continue
		}
		mountPointRequests[request.MountPoint] = request
	}
	return mountPointRequests
}
