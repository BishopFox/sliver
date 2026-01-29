## Contributing to Notify

We want to make contributing to this project as easy and transparent as possible.

## Tests

Ideally, unit tests should accompany every newly introduced exported function. We're always striving to increase the project's test coverage.

## Commits

Commit messages should be well formatted, and to make that "standardized", we are using Conventional Commits.

You can follow the documentation on [their website](https://www.conventionalcommits.org).

## Pull Requests

We actively welcome your pull requests.

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed or added exported functions or types, document them.
4. We use [gofumpt](https://github.com/mvdan/gofumpt) to format our code. Don't forget to always run `make fmt` before opening a new PR.
5. Ensure the test suite passes and the linter doesn't complain (`make ci`).


## Issues

We use GitHub issues to track public bugs. Please ensure your description is clear and has sufficient instructions to be
able to reproduce the issue.

## License

By contributing to notify, you agree that your contributions will be licensed under the LICENSE file in the root
directory of this source tree.
