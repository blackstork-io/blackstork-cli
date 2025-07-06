package engine

import (
	"testing"
)

func TestTags(t *testing.T) {
	renderTest(
		t, "basic",
		[]string{`
			document "test-doc" {
				content text {
					meta {
						tags = ["tag1"]
					}
					value = "tag1"
				}
				content text {
					meta {
						tags = ["tag2"]
					}
					value = "tag2"
				}
			}
		`},
		[]string{
			"tag2",
		},
		optRequiredTags{"tag2"},
	)
	renderTest(
		t, "document include all",
		[]string{`
			document "test-doc" {
				meta {
					tags = ["tag2"]
				}
				content text {
					meta {
						tags = ["tag1"]
					}
					value = "tag1"
				}
				content text {
					meta {
						tags = ["tag2"]
					}
					value = "tag2"
				}
			}
		`},
		[]string{
			"tag1",
			"tag2",
		},
		optRequiredTags{"tag2"},
	)
	renderTest(
		t, "section include all",
		[]string{`
			document "test-doc" {
				content text {
					value = "not included"
				}
				section {
					meta {
						tags = ["tag2"]
					}
					content text {
						meta {
							tags = ["tag1"]
						}
						value = "tag1"
					}
					content text {
						meta {
							tags = ["tag2"]
						}
						value = "tag2"
					}
				}
			}
		`},
		[]string{
			"tag1",
			"tag2",
		},
		optRequiredTags{"tag2"},
	)
	renderTest(
		t, "section children filtering",
		[]string{`
			document "test-doc" {
				content text {
					meta {
						tags = ["tag2"]
					}
					value = "included"
				}
				section {
					content text {
						meta {
							tags = ["tag1"]
						}
						value = "tag1"
					}
					content text {
						meta {
							tags = ["tag2"]
						}
						value = "tag2"
					}
				}
			}
		`},
		[]string{
			"included",
			"tag2",
		},
		optRequiredTags{"tag2"},
	)
	renderTest(
		t, "multiple tags",
		[]string{`
			document "test-doc" {
				content text {
					meta {
						tags = ["tag1"]
					}
					value = "1"
				}
				content text {
					meta {
						tags = ["tag1", "tag2"]
					}
					value = "12"
				}
				content text {
					meta {
						tags = ["tag1", "tag2", "tag3"]
					}
					value = "123"
				}
				content text {
					meta {
						tags = ["tag1", "tag2", "tag3", "tag4"]
					}
					value = "1234"
				}
			}
		`},
		[]string{
			"123",
			"1234",
		},
		optRequiredTags{"tag1", "tag2", "tag3"},
	)
}

func TestTagsWithDynamics(t *testing.T) {
	renderTest(
		t, "dynamics with tags",
		[]string{`
			document "test-doc" {
				dynamic {
					items = ["a", "b", "c"]
					content text {
						meta {
							tags = ["tag1"]
						}
						value = "tag1 {{.vars.dynamic_item_index}}: {{.vars.dynamic_item}}"
					}
					content text {
						meta {
							tags = ["tag2"]
						}
						value = "tag2 {{.vars.dynamic_item_index}}: {{.vars.dynamic_item}}"
					}
				}
			}
		`},
		[]string{
			"tag2 0: a",
			"tag2 1: b",
			"tag2 2: c",
		},
		optRequiredTags{"tag2"},
	)
}
