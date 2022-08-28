package main

import "testing"

func TestGetResourceTypeAndID(t *testing.T) {
	data := "project/foo/hoge/bar"
	resourceType := "project"
	resourceID := "foo/hoge/bar"

	if gotT, gotI, err := getResourceTypeAndID(data); err != nil {
		t.Error(err)
	} else if gotT != resourceType || gotI != resourceID {
		t.Fatalf("expected resource id %s and resource type %s, but got resource id %s and resource type %s", resourceID, resourceType, gotI, gotT)
	}
}
