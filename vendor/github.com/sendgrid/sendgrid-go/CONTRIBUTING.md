Hello! Thank you for choosing to help contribute to one of the Twilio SendGrid open-source libraries. There are many ways you can contribute and help is always welcome.  We simply ask that you follow the following contribution policies.

All third-party contributors acknowledge that any contributions they provide will be made under the same open-source license that the open-source project is provided under.

- [Improvements to the Codebase](#improvements-to-the-codebase)
- [Understanding the Code Base](#understanding-the-codebase)
- [Testing](#testing)
- [Style Guidelines & Naming Conventions](#style-guidelines-and-naming-conventions)
- [Creating a Pull Request](#creating-a-pull-request)
- [Code Reviews](#code-reviews)

There are a few ways to contribute, which we'll enumerate below:

<a name="improvements-to-the-codebase"></a>
## Improvements to the Codebase

We welcome direct contributions to the sendgrid-go code base. Thank you!

### Development Environment ###

#### Install and Run Locally ####

##### Prerequisites #####

- Go version 1.14, 1.15 or 1.16
- [rest](https://github.com/sendgrid/rest)

##### Initial setup: #####

```bash
git clone https://github.com/sendgrid/sendgrid-go.git
cd sendgrid-go
```

### Environment Variables

First, get your free Twilio SendGrid account [here](https://sendgrid.com/free?source=sendgrid-go).

Next, update your environment with your [SENDGRID_API_KEY](https://app.sendgrid.com/settings/api_keys).

```bash
echo "export SENDGRID_API_KEY='YOUR_API_KEY'" > sendgrid.env
echo "sendgrid.env" >> .gitignore
source ./sendgrid.env
```

##### Execute: #####

* Check out the documentation for [Web API v3 endpoints](https://sendgrid.com/docs/API_Reference/Web_API_v3/index.html).
* Review the corresponding [example](examples).
* Update the file

```bash
go run <Desired File>
```

<a name="understanding-the-codebase"></a>
## Understanding the Code Base

**/examples**

Working examples that demonstrate usage.

**sendgrid.go**

The main function that does the heavy lifting (and external entry point) is `API`.

<a name="testing"></a>
## Testing

All PRs require passing tests before the PR will be reviewed. All test files are in [`sendgrid_test.go`](sendgrid_test.go). For the purposes of contributing to this repo, please update the [`sendgrid_test.go`](sendgrid_test.go) file with unit tests as you modify the code.

The integration tests require a Twilio SendGrid mock API in order to execute. We've simplified setting this up using Docker to run the tests. You will just need [Docker Desktop](https://docs.docker.com/get-docker/) and `make`.

Once these are available, simply execute the Docker test target to run all tests: `make test-docker`. This command can also be used to open an interactive shell into the container where this library is installed. To start a *bash* shell for example, use this command: `command=bash make test-docker`.

<a name="style-guidelines-and-naming-conventions"></a>
## Style Guidelines & Naming Conventions

Generally, we follow the style guidelines as suggested by the official language. However, we ask that you conform to the styles that already exist in the library. If you wish to deviate, please explain your reasoning.

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

Please run your code through:

- [fmt](https://blog.golang.org/go-fmt-your-code)

<a name="creating-a-pull-request"></a>
## Creating a Pull Request

1. [Fork](https://help.github.com/fork-a-repo/) the project, clone your fork,
   and configure the remotes:

   ```bash
   # Clone your fork of the repo into the current directory
   git clone https://github.com/sendgrid/sendgrid-go
   
   # Navigate to the newly cloned directory
   cd sendgrid-go
   
   # Assign the original repo to a remote called "upstream"
   git remote add upstream https://github.com/sendgrid/sendgrid-go
   ```

2. If you cloned a while ago, get the latest changes from upstream:

   ```bash
   git checkout <dev-branch>
   git pull upstream <dev-branch>
   ```

3. Create a new topic branch (off the main project development branch) to
   contain your feature, change, or fix:

   ```bash
   git checkout -b <topic-branch-name>
   ```

4. Commit your changes in logical chunks. Please adhere to these [git commit
   message guidelines](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)
   or your code is unlikely to be merged into the main project. Use Git's
   [interactive rebase](https://help.github.com/articles/interactive-rebase)
   feature to tidy up your commits before making them public.

4a. Create tests.

4b. Create or update the example code that demonstrates the functionality of this change to the code.

5. Locally merge (or rebase) the upstream development branch into your topic branch:

   ```bash
   git pull [--rebase] upstream main
   ```

6. Push your topic branch up to your fork:

   ```bash
   git push origin <topic-branch-name>
   ```

7. [Open a Pull Request](https://help.github.com/articles/using-pull-requests/)
    with a clear title and description against the `main` branch. All tests must be passing before we will review the PR.

<a name="code-reviews"></a>
## Code Reviews

If you can, please look at open PRs and review them. Give feedback and help us merge these PRs much faster! If you don't know how, Github has some [great information on how to review a Pull Request](https://help.github.com/articles/about-pull-request-reviews/).
