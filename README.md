update archive/zip

### updates

zip.Writer can write zip.Reader's File as it is.

```go
r, _ := zip.NewReader(inputReader, inputSize)
w, _ := zip.NewWriter(outputWriter)

for _, file := range r.File {
    // if you don't want a data descriptor,
    // you need unset data descriptor flag.
    file.Flags &= ^FlagDataDescriptor

    // copy zip entry
    w.CopyFile(file)
}
```
