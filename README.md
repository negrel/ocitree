# :evergreen_tree: `ocitree` - Manage OCI/Docker images as git repositories.

![push workflow](https://github.com/negrel/ocitree/actions/workflows/push.yaml/badge.svg)

`ocitree` is a tool to manipulate OCI/Docker images using git like commands (`rebase`, `log`, `checkout`, etc).
It is inspired by `ostree` but uses OCI/Docker images as storage.

## Getting started

Let's start by installing `ocitree`.

### Installation

Currently, you can only install ocitree using `go install`:

```shell
go install github.com/negrel/ocitree@latest
```

### Usage

`ocitree` is designed to be similar to git, you can see the list of command by 
executing the following command:

```shell
# Print help informations
ocitree --help
```

## TODO

- [ ] Rebase user changes
	- [x] pick rebase choice
	- [ ] exec rebase choice
	- [x] drop rebase choice
	- [ ] reword rebase choice
	- [ ] squash rebase choice

## Contributing

If you want to contribute to `ocitree` to add a feature or improve the code contact
me at [negrel.dev@protonmail.com](mailto:negrel.dev@protonmail.com), open an
[issue](https://github.com/negrel/ocitree/issues) or make a
[pull request](https://github.com/negrel/ocitree/pulls).

## :stars: Show your support

Please give a :star: if this project helped you!

[![buy me a coffee](.github/images/bmc-button.png)](https://www.buymeacoffee.com/negrel)

## :scroll: License

MIT © [Alexandre Negrel](https://www.negrel.dev/)
