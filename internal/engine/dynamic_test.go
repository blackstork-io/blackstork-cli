package engine

import (
	"testing"

	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
)

func TestDynamic(t *testing.T) {
	renderTest(
		t, "dynamic content",
		[]string{`
			document "test-doc" {
				dynamic {
					items = ["a", "b", "c"]
					content text {
						value = "{{.vars.dynamic_item_index}}: {{.vars.dynamic_item}}"
					}
				}
			}
		`},
		[]string{
			"0: a",
			"1: b",
			"2: c",
		},
	)

	renderTest(
		t, "dynamic items is_included",
		[]string{`
			document "test-doc" {
				vars {
				   show1 = true
				   show2 = false
				}
				dynamic {
					items = ["dyn_item"]
					content text {
						is_included = query_jq(".vars.show1")
						value = "show1 block {{.vars.dynamic_item}}"
					}
				}
				dynamic {
					items = ["dyn_item"]
					content text {
						is_included = query_jq(".vars.show2")
						value = "show2 block {{.vars.dynamic_item}}"
					}
				}
			}
		`},
		[]string{
			"show1 block dyn_item",
		},
	)
	renderTest(
		t, "is_included with nested dynamics",
		[]string{`
			document "test-doc" {
				vars {
				   show1 = true
				   show2 = false
				}
				section {
					is_included = query_jq(".vars.show1")
					dynamic {
						items = ["dyn_item"]
						content text {
							value = "show1 block {{.vars.dynamic_item}}"
						}
					}
				}
				section {
					is_included = query_jq(".vars.show2")
					dynamic {
						items = ["dyn_item"]
						content text {
							value = "show2 block {{.vars.dynamic_item}}"
						}
					}
				}
			}
		`},
		[]string{
			"show1 block dyn_item",
		},
	)

	renderTest(
		t, "nested dynamics",
		[]string{`
			document "test-doc" {
				dynamic {
					items = ["abc", "def"]
					content text {
						value = "{{.vars.dynamic_item}} by letters:"
					}
					dynamic {
						items = query_jq(".vars.dynamic_item | split(\"\")")
						content text {
							vars {
								idx_human = query_jq(".vars.dynamic_item_index + 1")
							}
							value = "{{.vars.idx_human}}: {{.vars.dynamic_item}}"
						}
					}
				}
			}
		`},
		[]string{
			"abc by letters:",
			"1: a",
			"2: b",
			"3: c",
			"def by letters:",
			"1: d",
			"2: e",
			"3: f",
		},
	)
	renderTest(
		t, "dynamics sections and nested dynamic",
		[]string{`
			document "test-doc" {
				dynamic {
					items = ["A", "B"]
					section {
						content text {
							value = query_jq("\"Section \" + .vars.dynamic_item")
						}
						dynamic {
							items = ["x", "y"]
							content text {
								value = "Content {{.vars.dynamic_item}}"
							}
						}
					}
				}
			}
		`},
		[]string{
			"Section A",
			"Content x",
			"Content y",
			"Section B",
			"Content x",
			"Content y",
		},
	)
	renderTest(
		t, "dynamics sections with titles",
		[]string{`
			document "test-doc" {
				dynamic {
					items = ["A", "B"]
					section {
						title = query_jq("\"Section \" + .vars.dynamic_item")
						dynamic {
							items = ["x", "y"]
							content text {
								value = "Content {{.vars.dynamic_item}}"
							}
						}
					}
				}
			}
		`},
		[]string{
			// TODO: Fix titles after merging Ast
			"# Section A",
			"Content x",
			"Content y",
			"# Section B",
			"Content x",
			"Content y",
		},
	)
	renderTest(
		t, "dynamic refs",
		[]string{`
			content text "test" {
				value = "test {{.vars.dynamic_item}}"
			}
			document "test-doc" {
				dynamic {
					items = ["A", "B"]
					section {
						title = query_jq("\"Section \" + .vars.dynamic_item")
						content ref {
							base = content.text.test
						}
					}
				}
			}
		`},
		[]string{
			// TODO: Fix titles after merging Ast
			"# Section A",
			"test A",
			"# Section B",
			"test B",
		},
	)
	renderTest(
		t, "dynamic section ref",
		[]string{`
			section "test" {
				title = query_jq("\"Section \" + .vars.dynamic_item")
				content text {
					value = "test {{.vars.dynamic_item}}"
				}
			}
			document "test-doc" {
				dynamic {
					items = ["A", "B"]
					section ref {
						base = section.test
					}
					content text {
						value = "test2 {{.vars.dynamic_item}}"
					}
				}
			}
		`},
		[]string{
			// TODO: Fix titles after merging Ast
			"# Section A",
			"test A",
			"test2 A",
			"# Section B",
			"test B",
			"test2 B",
		},
	)
	renderTest(
		t, "dynamic with nested is_included",
		[]string{`
			document "test-doc" {
				dynamic {
					items = ["A", "B"]
					content text {
						value = "test {{.vars.dynamic_item}}"
					}
					section {
						title = query_jq("\"Section \" + .vars.dynamic_item")
						content text {
							is_included = query_jq(".vars.dynamic_item == \"B\"")
							value = "only for B: {{.vars.dynamic_item_index}} {{.vars.dynamic_item}}"
						}
					}
				}
			}
		`},
		[]string{
			// TODO: Fix titles after merging Ast
			"test A",
			"# Section A",
			"test B",
			"# Section B",
			"only for B: 1 B",
		},
	)
	renderTest(
		t, "dynamic with immediate nested is_included",
		[]string{`
			document "test-doc" {
				dynamic {
					items = ["A", "B"]
					content text {
						value = "test {{.vars.dynamic_item}}"
					}
					content text {
						is_included = query_jq(".vars.dynamic_item == \"B\"")
						value = "only for B: {{.vars.dynamic_item_index}} {{.vars.dynamic_item}}"
					}
				}
			}
		`},
		[]string{
			"test A",
			"test B",
			"only for B: 1 B",
		},
	)
	renderTest(
		t, "redefined dynamics",
		[]string{`
			document "test-doc" {
				dynamic {
					items = ["abc", "def"]
					content text {
						vars {
							// overrides only locally, does not affect other dynamic blocks
							dynamic_item = "XYZ"
						}
						value = "{{.vars.dynamic_item}} by letters:"
					}
					dynamic {
						items = query_jq(".vars.dynamic_item | split(\"\")")
						content text {
							vars {
								idx_human = query_jq(".vars.dynamic_item_index + 1")
							}
							value = "{{.vars.idx_human}}: {{.vars.dynamic_item}}"
						}
					}
				}
			}
		`},
		[]string{
			"XYZ by letters:",
			"1: a",
			"2: b",
			"3: c",
			"XYZ by letters:",
			"1: d",
			"2: e",
			"3: f",
		},
	)
	// TODO: Fix redefined inner dynamics with async evaluators
	// renderTest(
	// 	t, "redefined inner dynamics",
	// 	[]string{`
	// 		document "test-doc" {
	// 			dynamic {
	// 				items = ["abc", "def"]
	// 				section {
	// 					vars {
	// 						dynamic_item = "XYZ"
	// 					}
	// 					content text {
	// 						value = "{{.vars.dynamic_item}} by letters:"
	// 					}
	// 					dynamic {
	// 						items = query_jq(".vars.dynamic_item | split(\"\")")
	// 						content text {
	// 							vars {
	// 								idx_human = query_jq(".vars.dynamic_item_index + 1")
	// 							}
	// 							value = "{{.vars.idx_human}}: {{.vars.dynamic_item}}"
	// 						}
	// 					}
	// 				}
	// 			}
	// 		}
	// 	`},
	// 	[]string{
	// 		"XYZ by letters:",
	// 		"1: X",
	// 		"2: Y",
	// 		"3: Z",
	// 		"XYZ by letters:",
	// 		"1: X",
	// 		"2: Y",
	// 		"3: Z",
	// 	},
	// )
	// TODO: Fix deep nested dynamics with async evaluators
	// renderTest(
	// 	t, "deeply nested dynamics",
	// 	[]string{`
	// 		document "test-doc" {
	// 			dynamic {
	// 				items = ["a", "b", "c"]
	// 				content text {
	// 					value = "1. {{.vars.dynamic_item}}"
	// 				}
	// 				section {
	// 					is_included = query_jq(".vars.dynamic_item != \"b\"")
	// 					content text {
	// 						value = "2. {{.vars.dynamic_item}}"
	// 					}
	// 					section {
	// 						content text {
	// 							value = "3. {{.vars.dynamic_item}}"
	// 						}
	// 						dynamic {
	// 							items = query_jq("[.vars.dynamic_item, \"XYZ\", \"foo\"]")
	// 							section {
	// 								is_included = query_jq(".vars.dynamic_item_index != 0")
	// 								content text {
	// 									value = "4. {{.vars.dynamic_item}}"
	// 								}
	// 								content text {
	// 									is_included = query_jq(".vars.dynamic_item != \"foo\"")
	// 									value = "5. {{.vars.dynamic_item}}"
	// 								}
	// 							}
	// 						}
	// 					}
	// 				}
	// 			}
	// 		}
	// 	`},
	// 	[]string{
	// 		"1. a",
	// 		"2. a",
	// 		"3. a",
	// 		"4. XYZ",
	// 		"5. XYZ",
	// 		"4. foo",
	// 		"1. b",
	// 		"1. c",
	// 		"2. c",
	// 		"3. c",
	// 		"4. XYZ",
	// 		"5. XYZ",
	// 		"4. foo",
	// 	},
	// )
	renderTest(
		t, "warn on empty",
		[]string{`
			document "test-doc" {
				dynamic {
					content text {
						value = "hello"
					}
				}
			}
		`},
		[]string{},
		diagtest.Asserts{{
			diagtest.IsError,
			diagtest.SummaryEquals("Dynamic block without items"),
		}},
	)
	renderTest(
		t, "warn on no children empty",
		[]string{`
			document "test-doc" {
				content text {
					value = "hello"
				}
				dynamic {
					items = ["a", "b"]
				}
			}
		`},
		[]string{
			"hello",
		},
		diagtest.Asserts{{
			diagtest.IsWarning,
			diagtest.SummaryContains("Dynamic block without content"),
		}},
	)
}
