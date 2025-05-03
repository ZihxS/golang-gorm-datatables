# jQuery DataTables API for Go with Gorm

The golang-gorm-datatables package is a Go library which provides support for jQuery DataTables server-side processing using Gorm. It contains functions for generating SQL queries for filtering, sorting and pagination, as well as functions for counting the total and filtered records.

## Requirements

- Go 1.24 or higher
- Gorm 1.26 or higher

## Installation

To install the Request package, run the following command:

```bash
go get github.com/ZihxS/golang-gorm-datatables
```

## Simple Usage

```go
package main

import (
  // ...

  "github.com/ZihxS/golang-gorm-datatables" // [👉🏼 FOCUS HERE]

  // ...
)

func main() {
  // ...

  type User struct {
    ID   int
    Name string
    Age  int
  }

  // ...

  // example using mux router
  r.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
    req, err := datatables.ParseRequest(r) // parse the request [👉🏼 FOCUS HERE]
    if err != nil {
      http.Error(w, fmt.Sprintf("Error processing request: %v", err), http.StatusInternalServerError)
      return
    }

    tx := db.Model(&User{}) // gorm query [👉🏼 FOCUS HERE]
    response, err := datatables.New(tx).Req(*req).Make() // make datatables [👉🏼 FOCUS HERE]
    if err != nil {
      http.Error(w, fmt.Sprintf("Error processing request: %v", err), http.StatusInternalServerError)
      return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(response)
  })
}

// ...
```

## More Example & Documentation

You can visit this link to see more example and documentation:
- Documentation: ?
- Example on Server Side: ?
- Example on Client Side: ?

## Contributing

Contributions are welcome! If you'd like to contribute to the Request package, please follow these steps:

1. Fork the repository to your own GitHub account.
2. Clone the repository to your local machine.
3. Run `make deps` to install dependencies.
4. Make changes to the code, following the guidelines below.
5. Run `make test` to ensure your changes pass the test suite.
6. Run `make fmt` to ensure your code is formatted correctly.
7. Run `make lint` to ensure your code lint is clean.
8. Commit your changes with a meaningful commit message.
9. Push your changes to your forked repository.
10. Submit a pull request to the original repository.

### Makefile Commands

The Makefile provides several commands to help with development and testing:

* `make deps`: Installs dependencies for the package.
* `make test`: Runs the test suite for the package.
* `make test-coverage`: Runs the test suite with code coverage analysis.
* `make view-coverage`: Opens the test coverage report in your web browser.
* `make fmt`: Formats the code according to the Go standard.
* `make lint`: Runs the linter to check for coding style issues.
* `make build`: Builds the package and its dependencies.
* `make clean`: Removes any build artifacts and temporary files.

### Guidelines

When contributing to the Request package, please follow these guidelines:

* Use the Go standard coding style.
* Write comprehensive tests for any new functionality.
* Keep commits small and focused on a single change.
* Use meaningful commit messages that describe the change.

## License

The Request package is licensed under the MIT License.

## Authors

* [Muhammad Saleh Solahudin](https://github.com/ZihxS)

## Contributors

<a href="https://github.com/ZihxS/golang-gorm-datatables/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=ZihxS/golang-gorm-datatables" />
</a>

## Credits

- [DataTables](https://datatables.net)
- [Gorm](https://gorm.io)
