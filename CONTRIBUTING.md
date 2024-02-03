# Contribution Guide

## Committing

When publishing commits, the commit message should be structured as follows:

```txt
<type>: <description>

[optional body]

[optional footer]

```

Acceptable types include:

**fix**: a commit of the type fix patches a bug in your codebase (this correlates with PATCH in semantic versioning).

**feat**: a commit of the type feat introduces a new feature to the codebase (this correlates with MINOR in semantic versioning).

**config**: a commit of the type config is related to a configuration/settings change.

**chore**: a commit of the type config is updating grunt tasks etc; no production code change

**docs**: a commit of the type docs is updating project documentation.

**style**: a commit of the type style is updating code stylization.

**lint**: a commit of the type lint is related to linting changes.

**refactor**: a commit of the type refactor is refactoring production code.

**perf**: a commit of the type perf is improves on the performance of code.

**test**: a commit of the type test is updating project test.

**BREAKING CHANGE**: a commit that has the text BREAKING CHANGE: at the beginning of its optional body or footer section introduces a breaking API change (correlating with MAJOR in semantic versioning). A breaking change can be part of commits of any type. e.g., a fix:, feat: & chore: types would all be valid, in addition to any other type.

See https://www.conventionalcommits.org/en/v1.0.0/

## Code Commenting

When commenting code, stick to the official Golang recommendation of using complete sentences that start with the object name. E.g.

```go
// Request represents a request to run a command.
type Request struct { ...

// Encode writes the JSON encoding of req to w.
func Encode(w io.Writer, req *Request) { ...
and so on.
```

See https://golang.org/doc/effective_go.html#commentary.

## Package and File Creation

### Package Names

All references to names in your package will be done using the package name, so you can omit that name from the identifiers. For example, if you are in package chubby, you don't need type ChubbyFile, which clients will write as chubby.ChubbyFile. Instead, name the type File, which clients will write as chubby.File. Avoid meaningless package names like util, common, misc, api, types, and interfaces. See http://golang.org/doc/effective_go.html#package-names and http://blog.golang.org/package-names for more.

### File Names

As with packages, keep file names simple and avoid meaningless file names i.e. it should be clear what type of utilities or functionality is located within a file. To this same point, ensure that code is segmented among file names and not sections in a single file.

### Encapsulation

Only expose functions and variables in packages that are intended to be used by consumers of a package. As an example, take the `pkg/tools/download` package. In this package, notice the only exposed functions are the `NewDownloader(...), Start(...), and Stop(...)` methods. Likewise, the only exposed structs are `File{} and Downloader{}`. All other functions and objects are not meant to be used by the package's consumers and as such, are not exposed to them.

## References

[Effective Go](https://golang.org/doc/effective_go.html#names)
[Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
