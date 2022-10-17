# Contributing to the experimental repo

Thank you for your interest in contributing!

This doc is about how to contribute to this repo specifically. For how to
contribute to tektoncd projects in general, see [the overview in our README](README.md)
and the individual `CONTRIBUTING.md` files in each respective project.

**All contributors must comply with
[the code of conduct](./code-of-conduct.md).**

PRs are welcome, and will follow
[the tektoncd pull request process](https://github.com/tektoncd/community/blob/master/process.md#pull-request-process).

## Adding a new project

Once [your experimental project proposal has been accepted](https://github.com/tektoncd/community/blob/main/process.md#proposing-projects):

- Create a new folder for your project
- Add an [OWNERS](https://github.com/tektoncd/community/blob/master/process.md#OWNERS) file only in the initial pull request so that the project owners can approve subsequent pull requests for your project
- Add a README describing your project
- Add your project [to the list of projects in presubmit-tests.sh](https://github.com/tektoncd/experimental/blob/main/test/presubmit-tests.sh#L61)
- Add a `test/presubmit-tests.sh` file to your project

## Code standards

There is no one-size-fits-all standard for code in this repo.
Projects are encouraged to define expectations in their own CONTRIBUTING.md documents,
such as whether code should include tests and whether it's OK for the same reviewer to
approve and LGTM. In general, to encourage experimentation, code in this repo is not held
to the same [standards](https://github.com/tektoncd/community/blob/main/standards.md#code)
as other repos in the tektoncd org.