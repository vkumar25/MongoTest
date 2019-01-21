package main
import (
"fmt"
"net/http"
)

func main(){
  http.HandleFunc("/hello", hello)
  http.ListenAndServe(":8080", nil)
  //fmt.Printf("hello world\n")
}

func hello(w http.ResponseWriter, r *http.Request){
  name := r.FormValue("name")
  if len(name) == 0 {
    name = "stranger"
  }
  fmt.Fprintf(w, "Hello %v", name)
}

