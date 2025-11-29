module github.com/xraph/authsome/examples/client-usage/go-example

go 1.25.3

require github.com/xraph/authsome/clients/go v0.0.0

// Use local generated client
replace github.com/xraph/authsome/clients/go => ../../../clients/go
