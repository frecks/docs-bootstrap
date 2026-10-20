---
type: how-to
---

# Install docs-bootstrap

Build the `docs-bootstrap` command and put it on your `PATH`, so you can set up and upgrade repositories from any directory. Each build includes a copy of the documentation system, so update it whenever the system changes.

## Before you begin

- Go 1.22 or later.
- Git, and access to the private `github.com/frecks/docs-bootstrap` repository.

## Install

1. Clone the repository:

   ```bash
   git clone git@github.com:frecks/docs-bootstrap.git
   ```

1. Build and install the command:

   ```bash
   cd docs-bootstrap
   go install .
   ```

   Go puts the command in the directory that `go env GOBIN` prints, or in the `bin` directory of `go env GOPATH` if `GOBIN` is empty, such as `~/go/bin`.

1. If that directory isn't on your `PATH`, add it in your shell's startup file, such as `~/.bashrc`. For `~/go/bin`, add:

   ```bash
   export PATH="$PATH:$HOME/go/bin"
   ```

## Verify

In a new terminal, run:

```bash
docs-bootstrap --version
```

The output shows the version, such as:

```text
docs-bootstrap 0.3.0
```

## Update

In your clone of the repository, pull the changes, and install the command again:

```bash
git pull
go install .
```

Then upgrade your repositories. See [Upgrade a repository](upgrade-a-repository.md).

## Troubleshooting

- **`docs-bootstrap: command not found`:** the install directory isn't on your `PATH`. Add the install directory to your `PATH`, as in step 3, and open a new terminal.

## Related

- [Set up a repository](set-up-a-repository.md)
- [Options](options.md)
