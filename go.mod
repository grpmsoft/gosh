module github.com/grpmsoft/gosh

go 1.25.1

require (
	github.com/google/uuid v1.6.0
	github.com/phoenix-tui/phoenix/clipboard v0.0.0-00010101000000-000000000000
	github.com/phoenix-tui/phoenix/components v0.0.0-00010101000000-000000000000
	github.com/phoenix-tui/phoenix/style v0.0.0-00010101000000-000000000000
	github.com/phoenix-tui/phoenix/tea v0.1.0-alpha.0
	github.com/stretchr/testify v1.11.1
	github.com/unilibs/uniwidth v0.0.0-00010101000000-000000000000
	golang.org/x/term v0.36.0
	mvdan.cc/sh/v3 v3.12.0
)

require (
	github.com/charmbracelet/x/ansi v0.10.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/phoenix-tui/phoenix/core v0.0.0-00010101000000-000000000000 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sys v0.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Phoenix TUI local development
replace github.com/phoenix-tui/phoenix/tea => ../tui/tea

replace github.com/phoenix-tui/phoenix/components => ../tui/components

replace github.com/phoenix-tui/phoenix/core => ../tui/core

replace github.com/phoenix-tui/phoenix/style => ../tui/style

replace github.com/phoenix-tui/phoenix/mouse => ../tui/mouse

replace github.com/phoenix-tui/phoenix/clipboard => ../tui/clipboard

// Uniwidth local development
replace github.com/unilibs/uniwidth => ../uniwidth
