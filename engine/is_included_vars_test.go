package engine

import "testing"

func TestSectionLevelVars(t *testing.T) {
	renderTest(
		t, "section level vars usage",
		[]string{`
			document "test1" {
				section {
					title = "Section with Local Vars"
					
					// These vars should be accessible within the section's content
					vars {
						section_message = "This message is from section vars"
					}
					
					content text {
						value = "{{.vars.section_message}}"
					}
				}
				
				section {
					title = "Section with Document and Local Vars"
					
					vars {
						local_message = "Local section message"
					}
					
					content text {
						value = "Local: {{.vars.local_message}}"
					}
				}
			}
		`},
		[]string{
			"# Section with Local Vars",
			"This message is from section vars",
			"# Section with Document and Local Vars",
			"Local: Local section message",
		},
		optDocName("test1"),
	)
}

func TestSectionLevelVarsInIsIncluded(t *testing.T) {
	// This test verifies that section-level variables are accessible in the is_included attribute
	renderTest(
		t, "section level vars in is_included",
		[]string{`
			document "test1" {
				section {
					title = "Section with Local Vars"
					
					// Define section-level vars
					vars {
						should_show = true
					}
					
					// Use the section-level var in is_included
					is_included = query_jq(".vars.should_show")
					
					content text {
						value = "This section should be visible"
					}
				}
				
				section {
					title = "Section with Local Vars - Hidden"
					
					// Define section-level vars with should_show set to false
					vars {
						should_show = false
					}
					
					// Use the section-level var in is_included
					is_included = query_jq(".vars.should_show")
					
					content text {
						value = "This section should NOT be visible"
					}
				}
				
				section {
					title = "Always Visible Section"
					content text {
						value = "This section is always visible"
					}
				}
			}
		`},
		[]string{
			"# Section with Local Vars",
			"This section should be visible",
			"# Always Visible Section",
			"This section is always visible",
		},
		optDocName("test1"),
	)
}

func TestIsIncludedVarsSection(t *testing.T) {
	// This test verifies the original use case from the question works now
	renderTest(
		t, "original use case with section vars",
		[]string{`
			document "test1" {
				vars {
					included = true
				}

				section {
					title = "Foo Section"

					is_included = query_jq(".vars.included")

					content text {
						value = "Foo section text"
					}
				}

				section {
					vars {
						internal_included = true
					}

					title = "Bar Section"

					is_included = query_jq(".vars.internal_included")

					content text {
						value = "Bar section text"
					}
				}
			}
		`},
		[]string{
			"# Foo Section",
			"Foo section text",
			"# Bar Section",
			"Bar section text",
		},
		optDocName("test1"),
	)
}

func TestIsIncludedVarsSectionExclude(t *testing.T) {
	// This test verifies the original use case from the question works now
	renderTest(
		t, "exclude use case with section vars",
		[]string{`
			document "test1" {
				vars {
					included = true
				}

				section {
					title = "Foo Section"

					is_included = query_jq(".vars.included")

					content text {
						value = "Foo section text"
					}
				}

				section {
					vars {
						internal_included = false
					}

					title = "Bar Section"

					is_included = query_jq(".vars.internal_included")

					content text {
						value = "Bar section text"
					}
				}
			}
		`},
		[]string{
			"# Foo Section",
			"Foo section text",
		},
		optDocName("test1"),
	)
}
