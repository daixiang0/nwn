# NWN

NWN, a tool that help you show or cleanup any whitenoise.

## Download

```shell
$ go get github.com/daixiang0/nwn
```

## Usage

```shell
$ nwn -h
usage: gci [flags] [path ...]
  -d	display diffs instead of rewriting files
  -w	write result to (source) file instead of stdout
```

## Example

```shell
$ cat -A test
1$
2  $
   $
4$
$ go get github.com/daixiang0/nwn
$ nwn -w test
$ cat -A test
1$
2$
$
4$
```
