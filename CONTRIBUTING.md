# Contributing
Thanks for your interest in contributing to Acorn. Here are a few guidelines to help make you successful.


## Pull requests
Contributions to the code or documentation should be submitted as pull requests. A maintainer will review your PR and either approve, reject, or ask for changes. Once a PR is approved, it can be merged by a maintainer. Keep PRs scoped to a single functional change. For example, only fix one bug or implement a single enhancement. This will simplify the review and merge process.

If your PR is addressing a GitHub issue, reference it in your PR, but don't use GitHub's [auto-close keywords](https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue#linking-a-pull-request-to-an-issue-using-a-keyword). We use issues to drive our QA efforts and don't want the issue to auto-close when the PR is merged.

PRs have tests and other validations ran against them in CI. You can run the equivalent locally:
```
# Run tests
make test

# Run linting validation
make validate

# If you've made a change to CLI commands or flags, generate the corresponding docs changes:
make gen-docs
```
Note that the tests in the `./integration` package require a running Kubernetes cluster and will create things in that cluster.


## Commits
When your PR is ready for review and merging, ensure your commit messages and structure make sense.

Regarding the commit message: follow best practices. Here are two good resources for writing good commit messages:
- [How to Write a Git Commit Message](https://cbea.ms/git-commit/)
- [Git-scm.com's "Commit guildelines"](ttps://www.git-scm.com/book/en/v2/Distributed-Git-Contributing-to-a-Project#_commit_guidelines)

From the first resource, here are the seven rules of a great commit message:

1. Separate subject from body with a blank line
2. Limit the subject line to 50 characters
3. Capitalize the subject line
4. Do not end the subject line with a period
5. Use the imperative mood in the subject line ("Fix bug-X" instead of "Fixed bug-X" or "Fixes bug-X")
6. Wrap the body at 72 characters
7. Use the body to explain _what and why_ vs. _how_

Regarding commit structure: sometimes, a single commit is the best option. Other times, several commits is better. As the author, the choice is yours, but if you have a long series of commits from your iterative development, you should condense those down into as few commits as is logical.


## Signing off on your work
Please indicate that your contribution adheres to the following [Developer Certificate of Origin](https://developercertificate.org/) by signing-off on each of your commits.
```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

You can sign-off on a commit automatically by adding the -s flag, like so:
```
git commit -s
```
This will add the following as a footer to your commit message:
```
Signed-off-by: Joe Smith <joe.smith@email.com>
```

We have a CI check that will fail if your commits do not contain this sign-off footer.

