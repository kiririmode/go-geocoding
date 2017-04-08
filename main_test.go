package main

import "testing"

func TestConvert(t *testing.T) {

	data := []struct {
		desc     string
		input    float64
		expected string
	}{
		{"東京駅 緯度", 35.681298, "35°40'52.7\""},
		{"東京駅 軽度", 139.766247, "139°45'58.5\""},
	}

	for _, v := range data {
		actual := convert(v.input)
		if actual != v.expected {
			t.Errorf("%f is converted to %s, but expectation is %s", v.input, actual, v.expected)
		}
	}
}
