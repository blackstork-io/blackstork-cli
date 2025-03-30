package engine

import "testing"

func TestDependsOnWithRefBlocks(t *testing.T) {
	renderTest(
		t, "depends_on with ref blocks",
		[]string{`
			section "foo" {
			  title = "Foo title"

			  content text {
				value = "Foo text"
			  }
			}

			content text "baz" {
			  value = "Baz text"
			}

			document "test1" {
			  // Define reference blocks first
			  section ref "aaa" {
				base = section.foo
				title = "Foo overload 1"
			  }

			  content ref "bbb" {
				base = content.text.baz
			  }
			  
			  // Then reference them in depends_on
			  content text {
				depends_on = [
				  "section.ref.aaa",
				  "content.ref.bbb",
				]
				value = "Doc content is ready"
			  }
			}
		`},
		[]string{
			"# Foo overload 1",
			"Foo text",
			"Baz text",
			"Doc content is ready",
		},
		optDocName("test1"),
	)

	renderTest(
		t, "depends_on before ref blocks",
		[]string{`
			section "foo" {
			  title = "Foo title"

			  content text {
				value = "Foo text"
			  }
			}

			content text "baz" {
			  value = "Baz text"
			}

			document "test1" {
			  // Reference blocks in depends_on before they're defined
			  content text {
				depends_on = [
				  "section.ref.aaa",
				  "content.ref.bbb",
				]
				value = "Doc content is ready"
			  }
			  
			  // Define reference blocks after depending on them
			  section ref "aaa" {
				base = section.foo
				title = "Foo overload 1"
			  }

			  content ref "bbb" {
				base = content.text.baz
			  }
			}
		`},
		[]string{
			"Doc content is ready",
			"# Foo overload 1",
			"Foo text",
			"Baz text",
		},
		optDocName("test1"),
	)
}
