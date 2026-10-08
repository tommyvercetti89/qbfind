//go:build windows

package main

import "testing"

func TestListviewMessageConstants(t *testing.T) {
	messages := []struct {
		name string
		got  int
		want int
	}{
		{"LVM_SETITEMCOUNT", LVM_SETITEMCOUNT, 0x102F},
		{"LVM_EDITLABELW", LVM_EDITLABELW, 0x1076},
		{"LVM_SETITEMSTATE", LVM_SETITEMSTATE, 0x102B},
		{"LVM_HITTEST", LVM_HITTEST, 0x1012},
		{"LVM_GETNEXTITEM", LVM_GETNEXTITEM, 0x100C},
		{"LVM_INSERTCOLUMNW", LVM_INSERTCOLUMNW, 0x1061},
		{"LVM_SETCOLUMNW", LVM_SETCOLUMNW, 0x1060},
		{"LVM_SETEXTENDEDLISTVIEWSTYLE", LVM_SETEXTENDEDLISTVIEWSTYLE, 0x1036},
		{"LVM_GETHEADER", LVM_GETHEADER, 0x101F},
		{"LVM_SETIMAGELIST", LVM_SETIMAGELIST, 0x1003},
		{"LVM_SETBKCOLOR", LVM_SETBKCOLOR, 0x1001},
		{"LVM_SETTEXTCOLOR", LVM_SETTEXTCOLOR, 0x1024},
		{"LVM_SETTEXTBKCOLOR", LVM_SETTEXTBKCOLOR, 0x1026},
	}
	for _, c := range messages {
		if c.got != c.want {
			t.Errorf("%s = %#x, want %#x", c.name, c.got, c.want)
		}
	}

	notifications := []struct {
		name string
		got  uint32
		want uint32
	}{
		{"LVN_GETDISPINFO", LVN_GETDISPINFO, 0xFFFFFF4F},
		{"LVN_BEGINLABELEDIT", LVN_BEGINLABELEDIT, 0xFFFFFF51},
		{"LVN_ENDLABELEDIT", LVN_ENDLABELEDIT, 0xFFFFFF50},
		{"LVN_COLUMNCLICK", LVN_COLUMNCLICK, 0xFFFFFF94},
	}
	for _, c := range notifications {
		if c.got != c.want {
			t.Errorf("%s = %#x, want %#x", c.name, c.got, c.want)
		}
	}
}
