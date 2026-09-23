package app

import (
	"reflect"
	"strings"
	"testing"
)

func TestPanelImagesKeepIDsInUploadOrder(t *testing.T) {
	store := &PanelImages{Directory: t.TempDir()}
	if store.Random("1191") != "" {
		t.Fatal("a character without pictures has one")
	}
	for range 3 {
		if err := store.Add("1191", pngFixture(t)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Add("1191", []byte("not a picture at all")); err == nil {
		t.Fatal("a file that is no picture was kept")
	}
	if err := store.Add("../1191", pngFixture(t)); err == nil {
		t.Fatal("a character outside the store was accepted")
	}
	names, err := store.List("1191")
	if err != nil || len(names) != 3 {
		t.Fatalf("names = %v %v", names, err)
	}
	if path := store.Random("1191"); !strings.HasPrefix(path, "panel-images/1191/") || !strings.HasSuffix(path, ".png") {
		t.Fatalf("random = %s", path)
	}
	// IDs follow the upload order; unknown and repeated IDs fail.
	removed, failed, err := store.Remove("1191", []string{"2", "2", "9", "x"})
	if err != nil || !reflect.DeepEqual(removed, []string{"2"}) || !reflect.DeepEqual(failed, []string{"2", "9", "x"}) {
		t.Fatalf("removed %v failed %v %v", removed, failed, err)
	}
	left, _ := store.List("1191")
	if !reflect.DeepEqual(left, []string{names[0], names[2]}) {
		t.Fatalf("left %v of %v", left, names)
	}
}
