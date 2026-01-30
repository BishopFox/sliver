# Contributing | Firebase Admin Go SDK

Thank you for contributing to the Firebase community!

 - [Have a usage question?](#question)
 - [Think you found a bug?](#issue)
 - [Have a feature request?](#feature)
 - [Want to submit a pull request?](#submit)
 - [Need to get set up locally?](#local-setup)


## <a name="question"></a>Have a usage question?

We get lots of those and we love helping you, but GitHub is not the best place for them. Issues
which just ask about usage will be closed. Here are some resources to get help:

- Go through the [guides](https://firebase.google.com/docs/admin/setup/)
- Read the full [API reference](https://godoc.org/firebase.google.com/go)

If the official documentation doesn't help, try asking a question on the
[Firebase Google Group](https://groups.google.com/forum/#!forum/firebase-talk/) or one of our
other [official support channels](https://firebase.google.com/support/).

**Please avoid double posting across multiple channels!**


## <a name="issue"></a>Think you found a bug?

Yeah, we're definitely not perfect!

Search through [old issues](https://github.com/firebase/firebase-admin-go/issues) before
submitting a new issue as your question may have already been answered.

If your issue appears to be a bug, and hasn't been reported,
[open a new issue](https://github.com/firebase/firebase-admin-go/issues/new). Please use the
provided bug report template and include a minimal repro.

If you are up to the challenge, [submit a pull request](#submit) with a fix!


## <a name="feature"></a>Have a feature request?

Great, we love hearing how we can improve our products! Share you idea through our
[feature request support channel](https://firebase.google.com/support/contact/bugs-features/).


## <a name="submit"></a>Want to submit a pull request?

Sweet, we'd love to accept your contribution!
[Open a new pull request](https://github.com/firebase/firebase-admin-go/pull/new/master) and fill
out the provided template.

Make sure to create all your pull requests against the `dev` branch. All development
work takes place on this branch, while the `master` branch is dedicated for released
stable code. This enables us to review and merge routine code changes, without
impacting downstream applications that are building against our `master`
branch.

**If you want to implement a new feature, please open an issue with a proposal first so that we can
figure out if the feature makes sense and how it will work.**

Make sure your changes pass our linter and the tests all pass on your local machine.
Most non-trivial changes should include some extra test coverage. If you aren't sure how to add
tests, feel free to submit regardless and ask us for some advice.

Finally, you will need to sign our
[Contributor License Agreement](https://cla.developers.google.com/about/google-individual),
and go through our code review process before we can accept your pull request.

### Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution.
This simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

### Code reviews

All submissions, including submissions by project members, require review. We
use GitHub pull requests for this purpose. Consult
[GitHub Help](https://help.github.com/articles/about-pull-requests/) for more
information on using pull requests.

## <a name="local-setup"></a>Need to get set up locally?

### Initial Setup

Use the standard GitHub and [Go development tools](https://golang.org/doc/cmd)
to build and test the Firebase Admin SDK. Follow the instructions given in
the [golang documentation](https://golang.org/doc/code.html) to get your
`GOPATH` set up correctly. Then execute the following series of commands
to checkout the sources of Firebase Admin SDK, and its dependencies:

```bash
$ cd $GOPATH
$ git clone https://github.com/firebase/firebase-admin-go.git src/firebase.google.com/go
$ go get -d -t firebase.google.com/go/... # Install dependencies
```

### Unit Testing

Invoke the `go test` command as follows to build and run the unit tests:

```bash
go test -test.short firebase.google.com/go/...
```

Note the `-test.short` flag passed to the `go test` command. This will skip
the integration tests, and only execute the unit tests.

### Integration Testing

Integration tests are executed against a real life Firebase project. If you do not already
have one suitable for running the tests against, you can create a new project in the
[Firebase Console](https://console.firebase.google.com) following the setup guide below.
If you already have a Firebase project, you'll need to obtain credentials to communicate and
authorize access to your Firebase project:


1. Service account certificate: This allows access to your Firebase project through a service account
which is required for all integration tests. This can be downloaded as a JSON file from the 
**Settings > Service Accounts** tab of the Firebase console when you click the
**Generate new private key** button. Copy the file into the repo so it's available at
`src/firebase.google.com/go/testdata/integration_cert.json`.
   > **Note:** Service accounts should be carefully managed and their keys should never be stored in publicly accessible source code or repositories.


2. Web API key: This allows for Auth sign-in needed for some Authentication and Tenant Management
integration tests. This is displayed in the **Settings > General** tab of the Firebase console
after enabling Authentication as described in the steps below. Copy it and save to a new text
file at `src/firebase.google.com/go/testdata/integration_apikey.txt`.


Set up your Firebase project as follows:


1. Enable Authentication:
   1. Go to the Firebase Console, and select **Authentication** from the **Build** menu.
   2. Click on **Get Started**.
   3. Select **Sign-in method > Add new provider > Email/Password** then enable both the
   **Email/Password** and **Email link (passwordless sign-in)** options.


2. Enable Firestore:
   1. Go to the Firebase Console, and select **Firestore Database** from the **Build** menu.
   2. Click on the **Create database** button. You can choose to set up Firestore either in
   the production mode or in the test mode.


3. Enable Realtime Database:
   1. Go to the Firebase Console, and select **Realtime Database** from the **Build** menu.
   2. Click on the **Create Database** button. You can choose to set up the Realtime Database
   either in the locked mode or in the test mode.

   > **Note:** Integration tests are not run against the default Realtime Database reference and are
   instead run against a database created at `https://{PROJECT_ID}.firebaseio.com`.
   This second Realtime Database reference is created in the following steps.

   3. In the **Data** tab click on the kebab menu (3 dots) and select **Create Database**.
   4. Enter your Project ID (Found in the **General** tab in **Account Settings**) as the
   **Realtime Database reference**. Again, you can choose to set up the Realtime Database
   either in the locked mode or in the test mode.


4. Enable Storage:
   1. Go to the Firebase Console, and select **Storage** from the **Build** menu.
   2. Click on the **Get started** button. You can choose to set up Cloud Storage
   either in the production mode or in the test mode.


5. Enable the IAM API:
   1. Go to the [Google Cloud console](https://console.cloud.google.com)
   and make sure your Firebase project is selected.
   2. Select **APIs & Services** from the main menu, and click the
   **ENABLE APIS AND SERVICES** button.
   3. Search for and enable **Identity and Access Management (IAM) API** by Google Enterprise API.


6. Enable Tenant Management:
   1. Go to
   [Google Cloud console | Identity Platform](https://console.cloud.google.com/customer-identity/)
   and if it is not already enabled, click **Enable**.
   2. Then
   [enable multi-tenancy](https://cloud.google.com/identity-platform/docs/multi-tenancy-quickstart#enabling_multi-tenancy)
   for your project.


7. Ensure your service account has the **Firebase Authentication Admin** role. This is required
to ensure that exported user records contain the password hashes of the user accounts:
   1. Go to [Google Cloud console | IAM & admin](https://console.cloud.google.com/iam-admin).
   2. Find your service account in the list. If not added click the pencil icon to edit its
   permissions.
   3. Click **ADD ANOTHER ROLE** and choose **Firebase Authentication Admin**.
   4. Click **SAVE**.


Now you can invoke the test suite as follows:

```bash
go test firebase.google.com/go/...
```

This will execute both unit and integration test suites.

### Test Coverage

Coverage can be measured per package by passing the `-cover` flag to the test invocation:

```bash
go test -cover firebase.google.com/go/auth
```

To view the detailed coverage reports (per package):

```bash
go test -cover -coverprofile=coverage.out firebase.google.com/go
go tool cover -html=coverage.out
```
