update archive/zip

### updates

zip.Writer can write zip.Reader's File as it is.

```go
r, _ := zip.NewReader(inputReader, inputSize)
w, _ := zip.NewWriter(outputWriter)

for _, file := range r.File {
    w.CopyFile(file)
}
```
