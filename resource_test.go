package main

import "testing"

func TestGetResourceTypeAndID(t *testing.T) {
	type want struct {
		resourceType string
		resourceID   string
	}

	type testCase struct {
		data string
		want want
	}

	tc := []testCase{
		{
			data: "foo/hoge:6",
			want: want{
				resourceType: "foo",
				resourceID:   "hoge:6",
			},
		},
		{
			data: "project/foo/hoge/bar",
			want: want{
				resourceType: "project",
				resourceID:   "foo/hoge/bar",
			},
		},
		{
			data: "secret:hoge/bar",
			want: want{
				resourceType: "secret",
				resourceID:   "hoge/bar",
			},
		},
	}

	for _, v := range tc {
		if gotT, gotI, err := getResourceTypeAndID(v.data); err != nil {
			t.Error(err)
		} else if gotT != v.want.resourceType || gotI != v.want.resourceID {
			t.Fatalf("expected resource id %s and resource type %s, but got resource id %s and resource type %s", v.want.resourceID, v.want.resourceType, gotI, gotT)
		}
	}
}
