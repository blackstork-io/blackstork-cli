package engine

import "testing"

func TestIsIncluded(t *testing.T) {
	renderTest(
		t, "content",
		[]string{`
			document "test-doc" {
				vars {
				   show1 = true
				   show2 = false
				}
				content text {
					is_included = false
					value = "show1 block"
				}
				content text {
					is_included = true
					value = "show2 block"
				}
				content text {
					value = "show3 block"
				}
			}
		`},
		[]string{
			"show2 block",
			"show3 block",
		},
	)
	renderTest(
		t, "content query",
		[]string{`
			document "test-doc" {
				vars {
				   show1 = true
				   show2 = false
				}
				content text {
					is_included = query_jq(".vars.show1")
					value = "show1 block"
				}
				content text {
					is_included = query_jq(".vars.show2")
					value = "show2 block"
				}
			}
		`},
		[]string{
			"show1 block",
		},
	)

	renderTest(
		t, "section",
		[]string{`
			document "test-doc" {
				vars {
				   show1 = true
				   show2 = false
				}
				section {
					is_included = query_jq(".vars.show1")
					content text {
						value = "show1 block"
					}
				}
				section {
					is_included = query_jq(".vars.show2")
					content text {
						value = "show2 block"
					}
				}
			}
		`},
		[]string{
			"show1 block",
		},
	)
	renderTest(
		t, "ref",
		[]string{`
			content text "test" {
				vars {
					text = "base value"
				}
				value = "test: {{.vars.text}}"
			}

			content text "test_not_included" {
				vars {
					text = "base value"
				}
				is_included = false
				value = "test_not_included: {{.vars.text}}"
			}
			section "test_section_not_included" {
				vars {
					text = "base value"
				}
				is_included = false
				content text {
					value = "test_section_not_included: {{.vars.text}}"
				}
			}

			document "test-doc" {
				content ref {
					is_included = false
					vars {
						text = "ref value1"
					}
					base = content.text.test
				}
				content ref {
					is_included = true
					vars {
						text = "ref value2"
					}
					base = content.text.test
				}
				content ref {
					vars {
						text = "ref value1"
					}
					base = content.text.test_not_included
				}
				content ref {
					vars {
						text = "ref value2"
					}
					is_included = true
					base = content.text.test_not_included
				}
				section ref {
					vars {
						text = "ref value1"
					}
					base = section.test_section_not_included
				}
				section ref {
					vars {
						text = "ref value2"
					}
					is_included = true
					base = section.test_section_not_included
				}
			}
		`},
		[]string{
			"test: ref value2",
			"test_not_included: ref value2",
			"test_section_not_included: ref value2",
		},
	)
	renderTest(
		t, "truthiness",
		[]string{`
			document "test-doc" {
				content text {
					is_included = []
					value = "falsy"
				}
				content text {
					is_included = {}
					value = "falsy"
				}
				content text {
					is_included = ""
					value = "falsy"
				}
				content text {
					is_included = query_jq("[]")
					value = "falsy"
				}
				content text {
					is_included = query_jq("{}")
					value = "falsy"
				}
				content text {
					is_included = query_jq("\"\"")
					value = "falsy"
				}
				content text {
					is_included = [1]
					value = "truthy"
				}
				content text {
					is_included = { a = "b"}
					value = "truthy"
				}
				content text {
					is_included = "a"
					value = "truthy"
				}
				content text {
					is_included = query_jq("[1]")
					value = "truthy"
				}
				content text {
					is_included = query_jq("{ \"a\": \"b\" }")
					value = "truthy"
				}
				content text {
					is_included = query_jq("\"hello\"")
					value = "truthy"
				}
			}
		`},
		[]string{
			"truthy",
			"truthy",
			"truthy",
			"truthy",
			"truthy",
			"truthy",
		},
	)
}
