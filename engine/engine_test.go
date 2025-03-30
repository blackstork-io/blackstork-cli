package engine

import (
	"log/slog"
	"os"
	"testing"

	"github.com/blackstork-io/fabric/pkg/diagnostics/diagtest"
	"github.com/blackstork-io/fabric/plugin/plugindata"
)

func TestEngineFetchData(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelWarn,
	})))
	fetchDataTest(
		t, "Basic",
		[]string{
			`
			document "hello" {
				data json "test" {
					path = "testdata/a.json"
				}

				data json "test2" {
					path = "testdata/a.json"
				}

				content text {
					value = "hello"
				}
			}
			`,
		},
		"document.hello.data.json.test",
		plugindata.Map{
			"data": plugindata.Map{
				"json": plugindata.Map{
					"test": plugindata.Map{
						"property_for": plugindata.String("a.json"),
					},
				},
			},
		},
		[][]diagtest.Assert{},
	)
	fetchDataTest(
		t, "Basic",
		[]string{
			`
			document "hello" {
				data json "test" {
					path = "testdata/a.json"
				}

				data json "test2" {
					path = "testdata/a.json"
				}

				content text {
					value = "hello"
				}
			}
			`,
		},
		"document.hello.data.json",
		plugindata.Map{
			"data": plugindata.Map{
				"json": plugindata.Map{
					"test": plugindata.Map{
						"property_for": plugindata.String("a.json"),
					},
					"test2": plugindata.Map{
						"property_for": plugindata.String("a.json"),
					},
				},
			},
		},
		[][]diagtest.Assert{},
	)
	fetchDataTest(
		t, "Basic",
		[]string{
			`
			data json "test" {
				path = "testdata/a.json"
			}
			document "hello" {
				content text {
					value = "hello"
				}
			}
			`,
		},
		"data.json.test",
		plugindata.Map{
			"property_for": plugindata.String("a.json"),
		},
		[][]diagtest.Assert{},
	)
}

func TestEngineLint(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelWarn,
	})))
	fullLintTest(
		t, "Ref loop",
		[]string{
			`
			document "test-doc" {
				content ref {
					base = content.ref.actual_block
				}
			}
			`,
			`
			content ref "ref_a" {
				base = content.ref.ref_b
			}

			content ref "ref_b" {
				base = content.ref.ref_c
			}

			content ref "ref_c" {
				base = content.ref.ref_a
			}

			content ref "actual_block" {
				base = content.ref.ref_a
			}
			`,
		},
		[][]diagtest.Assert{
			{diagtest.IsError, diagtest.SummaryContains("Circular reference detected")},
		},
	)
	fullLintTest(
		t, "Data ref name warning",
		[]string{
			`
			data json "name" {
				path = "testdata/a.json"
			}
			document "test-doc" {
				data ref {
					base = data.json.name
				}
				data ref {
					base = data.json.name
				}
			}
			`,
		},
		[][]diagtest.Assert{
			{diagtest.IsWarning, diagtest.SummaryContains("Data conflict")},
		},
	)

	limitedLintTest(
		t, "Unknown plugins are fine in limited lint",
		[]string{
			`
			document "doc1" {
				data made_up_data_source "name1" {

				}
			}
			document "doc2" {
				content made_up_content_provider "name2" {

				}
			}
			`,
		},
		[][]diagtest.Assert{},
	)
	fullLintTest(
		t, "Unknown plugins generate diags in full lint",
		[]string{
			`
			document "doc1" {
				data made_up_data_source "name1" {

				}
			}
			document "doc2" {
				content made_up_content_provider "name2" {

				}
			}
			`,
		},
		[][]diagtest.Assert{
			{diagtest.IsError, diagtest.SummaryContains("Missing datasource")},
			{diagtest.IsError, diagtest.SummaryContains("Missing content provider")},
		},
	)
	limitedLintTest(
		t, "Unknown config is fine in limited lint",
		[]string{
			`
			document "doc1" {
				data json "name1" {
					config {}
				}
			}
			`,
		},
		[][]diagtest.Assert{},
	)

	fullLintTest(
		t, "Unknown config generate diags in full lint",
		[]string{
			`
			document "doc1" {
				data json "name1" {
					config {}
				}
			}
			`,
		},
		[][]diagtest.Assert{
			{diagtest.IsWarning, diagtest.DetailContains("support configuration")},
		},
	)
}

func TestEnvPrefix(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelWarn,
	})))
	t.Setenv("OTHER_VAR", "OTHER_VAR")
	t.Setenv("FABRIC_VAR", "FABRIC_VAR")
	t.Setenv("FABRIC_TEST_VAR", "FABRIC_TEST_VAR")

	renderTest(
		t, "Default",
		[]string{
			`
			document "test-doc" {
				content text {
					value = "{{.env.OTHER_VAR}}\n{{.env.FABRIC_VAR}}\n{{.env.FABRIC_TEST_VAR}}"
				}
			}
			`,
		},
		[]string{
			"<no value>\nFABRIC_VAR\nFABRIC_TEST_VAR",
		},
	)
	renderTest(
		t, "Custom",
		[]string{
			`
			fabric {
				expose_env_vars_with_pattern = "FABRIC_TEST_*"
			}
			document "test-doc" {
				content text {
					value = "{{.env.OTHER_VAR}}\n{{.env.FABRIC_VAR}}\n{{.env.FABRIC_TEST_VAR}}"
				}
			}
			`,
		},
		[]string{
			"<no value>\n<no value>\nFABRIC_TEST_VAR",
		},
	)
	renderTest(
		t, "Empty",
		[]string{
			`
			fabric {
				expose_env_vars_with_pattern = ""
			}
			document "test-doc" {
				content text {
					value = "{{.env.OTHER_VAR}}\n{{.env.FABRIC_VAR}}\n{{.env.FABRIC_TEST_VAR}}"
				}
			}
			`,
		},
		[]string{
			"<no value>\n<no value>\n<no value>",
		},
	)
	renderTest(
		t, "Empty",
		[]string{
			`
			fabric {
				expose_env_vars_with_pattern =  "\t FABRIC_TEST_*   "
			}
			document "test-doc" {
				content text {
					value = "{{.env.OTHER_VAR}}\n{{.env.FABRIC_VAR}}\n{{.env.FABRIC_TEST_VAR}}"
				}
			}
			`,
		},
		[]string{
			"<no value>\n<no value>\nFABRIC_TEST_VAR",
		},
		diagtest.Asserts{{
			diagtest.IsWarning,
			diagtest.SummaryContains("contains a whitespace"),
		}},
	)
	renderTest(
		t, "ErrorInPattern",
		[]string{
			`
			fabric {
				expose_env_vars_with_pattern =  "FABRIC_TEST_["
			}
			document "test-doc" {
				content text {
					value = "{{.env.OTHER_VAR}}\n{{.env.FABRIC_VAR}}\n{{.env.FABRIC_TEST_VAR}}"
				}
			}
			`,
		},
		nil,
		diagtest.Asserts{{
			diagtest.IsError,
			diagtest.SummaryContains("Failed to parse", "expose_env_vars_with_pattern"),
		}},
	)
	renderTest(
		t, "Null",
		[]string{
			`
			fabric {
				expose_env_vars_with_pattern = null
			}
			document "test-doc" {
				content text {
					value = "{{.env.OTHER_VAR}}\n{{.env.FABRIC_VAR}}\n{{.env.FABRIC_TEST_VAR}}"
				}
			}
			`,
		},
		[]string{
			"<no value>\n<no value>\n<no value>",
		},
	)
}

func TestEngineRenderContent(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelWarn,
	})))
	renderTest(
		t, "Basic",
		[]string{
			`
			document "hello" {
				title = "Welcome"
				content text {
					value = "Hello from fabric"
				}
			}

			document "goodbye" {
				title = "Goodbye"
				content text {
					value = "Goodbye from fabric"
				}
			}
			`,
		},
		[]string{
			"# Welcome",
			"Hello from fabric",
		},
		optDocName("hello"),
	)
	renderTest(
		t, "Ref",
		[]string{
			`
			document "test-doc" {
				title = "Welcome"
				content ref {
					base = content.text.external_block
				}
			}

			content text "external_block" {
				value = "Hello from ref"
			}
			`,
		},
		[]string{
			"# Welcome",
			"Hello from ref",
		},
	)
	renderTest(
		t, "Ref across files",
		[]string{
			`
			document "test-doc" {
				title = "Welcome"
				content ref {
					base = content.text.external_block
				}
			}
			`,
			`
			content text "external_block" {
				value = "Hello from ref"
			}
			`,
		},
		[]string{
			"# Welcome",
			"Hello from ref",
		},
	)
	renderTest(
		t, "Ref chain",
		[]string{
			`
			document "test-doc" {
				content ref {
					base = content.ref.add_format_as
				}
			}

			content blockquote "actual_block" {
				value = "Hello from ref chain"
			}
			`,
			`
			content ref "add_format_as" {
				base = content.blockquote.actual_block
			}
			`,
		},
		[]string{
			"> Hello from ref chain",
		},
	)
	renderTest(
		t, "Ref loop untouched",
		[]string{
			`
			document "test-doc" {
				content ref {
					base = content.text.actual_block
				}
			}
			`,
			`
			content ref "ref_a" {
				base = content.ref.ref_b
			}

			content ref "ref_b" {
				base = content.ref.ref_c
			}

			content ref "ref_c" {
				base = content.ref.ref_a
			}

			content text "actual_block" {
				value = "Near refloop"
			}
			`,
		},
		[]string{
			"Near refloop",
		},
	)

	renderTest(
		t, "Ref loop touched",
		[]string{
			`
			document "test-doc" {
				content ref {
					base = content.ref.actual_block
				}
			}
			`,
			`
			content ref "ref_a" {
				base = content.ref.ref_b
			}

			content ref "ref_b" {
				base = content.ref.ref_c
			}

			content ref "ref_c" {
				base = content.ref.ref_a
			}

			content ref "actual_block" {
				base = content.ref.ref_a
			}
			`,
		},
		[]string{},
		diagtest.Asserts{
			{diagtest.IsError, diagtest.SummaryContains("Circular reference detected")},
		},
	)
	renderTest(
		t, "Sections",
		[]string{
			`
			section "sect4" {
				content text {
					value = "final section"
				}
			}
			document "test-doc" {
				section ref {
					base = section.sect1
				}
			}
			`,
			`
			section "sect1" {
				title = "sect1"
				content text {
					value = "s1"
				}
				content ref {
					base = content.text.some_text
				}
				section {
					title = "sect2"
					content ref {
						base = content.text.some_text
					}
					content text {
						value = "s2"
					}
					section {
						title = "sect3"
						content text {
							value = "s3"
						}
						content ref {
							base = content.text.some_text
						}
						content text {
							value = "s3 extra"
						}
						section ref {
							base = section.sect4
						}
					}
				}
			}

			content text "some_text" {
				value = "some_text"
			}

			`,
		},
		[]string{
			// TODO: Fix section title rendering with new Ast formatting
			"# sect1",
			"s1",
			"some_text",
			"# sect2",
			"some_text",
			"s2",
			"# sect3",
			"s3",
			"some_text",
			"s3 extra",
			"final section",
		},
	)
	renderTest(
		t, "Templating support",
		[]string{
			`
			document "test-doc" {
				content text {
					value = "${2+2}"
				}
			}
			`,
		},
		[]string{
			"4",
		},
	)
	renderTest(
		t, "Data ref name warning missing",
		[]string{
			`
			data json "name" {
				path = "testdata/a.json"
			}
			document "test-doc" {
				data ref {
					base = data.json.name
				}
			}
			`,
		},
		[]string{},
		diagtest.Asserts{
			{diagtest.IsWarning, diagtest.SummaryContains("No content")},
		},
	)
	renderTest(
		t, "Data ref name warning",
		[]string{
			`
			data json "name" {
				path = "testdata/a.json"
			}
			document "test-doc" {
				data ref {
					base = data.json.name
				}
				data ref {
					base = data.json.name
				}
			}
			`,
		},
		[]string{},
		diagtest.Asserts{
			{diagtest.IsWarning, diagtest.SummaryContains("Data conflict")},
			{diagtest.IsWarning, diagtest.SummaryContains("No content")},
		},
	)
	renderTest(
		t, "Content ref name no-error",
		[]string{
			`
			content text "name" {
				value = "txt"
			}
			document "test-doc" {
				content ref {
					base = content.text.name
				}
			}
			`,
		},
		[]string{"txt"},
	)
	renderTest(
		t, "No fabric files",
		[]string{},
		[]string{},
		diagtest.Asserts{
			{diagtest.IsError, diagtest.SummaryContains("No valid fabric files found")},
		},
	)
	renderTest(
		t, "Data block result access",
		[]string{
			`
			document "test-doc" {
				data json "name" {
					path = "testdata/a.json"
				}
				content text {
					value = "From data block: {{.data.json.name.property_for}}"
				}
			}
			`,
		},
		[]string{"From data block: a.json"},
	)
	renderTest(
		t, "Data with jq interaction",
		[]string{
			`
			document "test" {
				vars {
					items = ["a", "b", "c"]
					x = 1
					y = 2
				}
				content text {
					local_var = query_jq(".vars.items | length")
					value = "There are {{ .vars.local }} items"
				}
			}
			`,
		},
		[]string{"There are 3 items"},
		optDocName("test"),
	)
	renderTest(
		t, "Document meta",
		[]string{
			`
			document "test" {
				meta {
					authors = ["foo", "bar"]
					version = "0.1.2"
					tags = ["xxx", "yyy"]
				}
				content text {
					local_var = query_jq(".document.meta.authors")
					value = <<-EOT
						authors={{ .vars.local | join "," }},
						version={{ .document.meta.version }},
						tag={{ index .document.meta.tags 0 }}
					EOT
				}
			}
			`,
		},
		[]string{"authors=foo,bar,\nversion=0.1.2,\ntag=xxx"},
		optDocName("test"),
	)
	renderTest(
		t, "Document and content meta",
		[]string{
			`
			document "test" {
				meta {
					authors = ["foo"]
				}
				section {
					meta {
						authors = ["bar"]
					}
					content text {
						meta {
							authors = ["baz"]
						}
						local_var = query_jq(".document.meta.authors[0] + .section.meta.authors[0] + .content.meta.authors[0]")
						value = "author = {{ .vars.local }}"
					}
				}
			}
			`,
		},
		[]string{"author = foobarbaz"},
		optDocName("test"),
	)
	renderTest(
		t, "Meta scoping and nesting",
		[]string{
			`
			content text get_section_author {
				local_var = query_jq(".section.meta.authors[0] // \"unknown\"")
				value = "author = {{ .vars.local }}"
			}
			document "test" {
				content ref {
					base = content.text.get_section_author
				}
				section {
					content ref {
						base = content.text.get_section_author
					}
					section {
						meta {
							authors = ["foo"]
						}
						content ref {
							base = content.text.get_section_author
						}
						section {
							content ref {
								base = content.text.get_section_author
							}
							section {
								meta {
									authors = ["bar"]
								}
								content ref {
									base = content.text.get_section_author
								}
							}
						}
					}
				}
			}
			`,
		},
		[]string{
			"author = unknown",
			"author = unknown",
			"author = foo",
			"author = unknown",
			"author = bar",
		},
		optDocName("test"),
	)
	renderTest(
		t, "Reference rendered blocks",
		[]string{
			`
			document "test" {
				content text foo {
					value = "first result"
				}
				content text bar {
					depends_on = ["content.text.foo"]
					local_var = query_jq(".document.content.children[0].markdown")
					value = "content[0] = {{ .vars.local }}"
				}
				content text  baz{
					depends_on = ["content.text.bar"]
					local_var = query_jq(".document.content.children[1].markdown")
					value = "content[1] = {{ .vars.local }}"
				}
			}
			`,
		},
		[]string{
			"first result",
			"content[0] = first result",
			"content[1] = content[0] = first result",
		},
		optDocName("test"),
	)
}

func TestDependencyContextAccess(t *testing.T) {
	// Test case 1: Access dependency blocks via context
	renderTest(
		t, "Access dependency blocks via context",
		[]string{`
			document "test" {
				content text "foo" {
					value = "first result"
				}
				
				content text "bar" {
					depends_on = ["content.text.foo"]
					value = "second result"
				}
				
				content text "baz" {
					depends_on = ["content.text.foo", "content.text.bar"]
					value = "Dependency foo: {{ .dependency.content.text.foo.markdown }} and bar: {{ .dependency.content.text.bar.markdown }}"
				}
			}
		`},
		[]string{
			"first result",
			"second result",
			"Dependency foo: first result and bar: second result",
		},
		optDocName("test"),
	)

	// Test case 2: Access section dependency blocks via context
	renderTest(
		t, "Access section dependency blocks via context",
		[]string{`
			section "foo" {
				title = "Foo section"
				content text {
					value = "foo content"
				}
			}
			
			document "test" {
				section ref "bar" {
					base = section.foo
					title = "Bar section"
				}
				
				content text {
					depends_on = ["section.ref.bar"]
					value = <<-EOT
						Dependency: {{ .dependency | toPrettyJson }}
						Section title: {{ .dependency.section.ref.bar.meta.title }}
					EOT
				}
			}
		`},
		[]string{
			"# Bar section",
			"foo content",
			"Dependency: {\n" +
				"  \"section\": {\n" +
				"    \"ref\": {\n" +
				"      \"bar\": {\n" +
				"        \"meta\": {\n" +
				"          \"title\": \"foo\"\n" +
				"        }\n" +
				"      }\n" +
				"    }\n" +
				"  }\n" +
				"}\n" +
				"Section title: foo",
		},
		optDocName("test"),
	)
}
