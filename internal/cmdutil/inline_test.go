package cmdutil

import (
	"reflect"
	"testing"
)

func TestScanFileIDs(t *testing.T) {
	cases := []struct {
		name string
		html string
		want []int
	}{
		{"none", "<p>hello</p>", nil},
		{
			"class and src count once",
			`<img class="ckeditor-direct-upload-image-6340746" ` +
				`src="https://x/?action=getfile&uniqueid=6340746&planfixauth=t">`,
			[]int{6340746},
		},
		{
			"multiple distinct, ordered",
			`<img src="?uniqueid=11"> text <img src="?uniqueid=22">`,
			[]int{11, 22},
		},
		{
			"duplicates across the string deduped",
			`<img src="?uniqueid=11"><img src="?uniqueid=11"><img src="?uniqueid=22">`,
			[]int{11, 22},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ScanFileIDs(c.html)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("ScanFileIDs = %v, want %v", got, c.want)
			}
		})
	}
}
