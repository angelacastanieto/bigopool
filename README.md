# gopool

`gopool` is a small library that implements high performance worker pool in Golang and allows `error`/`result` handling in the main thread.

## Quickstart

install:
`go get -u github.com/bigodines/gopool`

implement this simple interface:
```golang
type TestJob {
    // your properties go here
}
func (j *TestJob) Execute() (gopool.Result, error) {
    // your logic here.
    // Result.Response is an interface{}
}
```

add to your code:
```golang
// configure dispatcher to run 5 workers with a queue of 100
dispatcher, err := gopool.NewDispatcher(5, 100)
if err != nil {
    panic(err)
}
// spawn workers
dispatcher.Run()
// send work items
dispatcher.Enqueue(TestJob{}) // <-- add one job
dispatcher.Enqueue(TestJob{}, TestJob{}) // <-- add multiple jobs

// wait for workers to finish (this is a blocking call)
dispatcher.Wait() 

errs := dispatcher.Errors
results := dispatcher.Results

```