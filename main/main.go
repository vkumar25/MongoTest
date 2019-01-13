package main
import(
  "encoding/json"
	"fmt"
  "log"
	"net/http"
  "goji.io"
  "goji.io/pat"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
)
const(
  mongoUrl = "localhost:27017"
)

func ErrorWithJSON(w http.ResponseWriter, message string, code int){
  w.Header().Set("Content-Type", "application/json;charset=utf-8")
  w.WriteHeader(code)
  fmt.Fprintf(w, "{message:%q}", message)
}

func ResponseWithJSON(w http.ResponseWriter, json []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(json)
}

type Book struct{
  ISBN string "json:isbn"
  Title string "json:name"
  Authors string "json:author"
  Price string "json:price"
}

/*
var bookstore = []book{
  book{
    ISBN:    "0321774639",
		Title:   "Programming in Go: Creating Applications for the 21st Century (Developer's Library)",
		Authors: "Mark Summerfield",
		Price:   "$34.57",
    },
    book{
      ISBN:    "0134190440",
		  Title:   "The Go Programming Language",
		  Authors: "Alan A. A. Donovan, Brian W. Kernighan",
		  Price:   "$34.57",
    },
}
*/


func main(){
  session, err := mgo.Dial("localhost")
  if err != nil {
    panic(err)
  }
  defer session.Close()
  session.SetMode(mgo.Monotonic, true)
  ensureIndex(session)

  mux := goji.NewMux()
  mux.HandleFunc(pat.Get("/books"), allBooks(session))
  mux.HandleFunc(pat.Post("/books"), addBook(session))
  mux.HandleFunc(pat.Get("/books/:isbn"),bookByISBN(session))
  mux.HandleFunc(pat.Put("/books/:isbn"), updateBook(session))
  mux.HandleFunc(pat.Delete("/books/:isbn"), deleteBook(session))
  http.ListenAndServe("localhost:3001", mux)
}

func ensureIndex(s *mgo.Session){
  session := s.Copy()
  defer session.Close()

  c := session.DB("store").C("books")
  index := mgo.Index{
    Key:                 []string {"isbn"},
    Unique:              true,
    DropDups:            true,
    Background:          true,
    Sparse:              true,
  }
  err := c.EnsureIndex(index)
  if err != nil {
    panic(err)
  }
}

func allBooks(s *mgo.Session) func(w http.ResponseWriter, r *http.Request){
  return func(w http.ResponseWriter, r *http.Request){
    session := s.Copy()
    defer session.Close()

    c := session.DB("store").C("books")

    var books []Book
    err := c.Find(bson.M{}).All(&books)
    if err != nil{
      ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
      log.Println("Failed get all books: ", err)
      return
    }

    respBody, err := json.MarshalIndent(books, "", " ")
    if err != nil{
      log.Fatal(err)
    }
    ResponseWithJSON(w, respBody, http.StatusOK)
  }
}

func addBook(s *mgo.Session) func(w http.ResponseWriter, r *http.Request){
  return func(w http.ResponseWriter, r *http.Request){
    session := s.Copy()
    defer session.Close()

    var book Book
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&book)
    if err != nil {
      ErrorWithJSON(w, "Incorrect request body", http.StatusBadRequest)
    }
    c := session.DB("store").C("books")
    err = c.Insert(book)
    if err != nil{
      if mgo.IsDup(err){
        ErrorWithJSON(w, "Book with ISBN already exists", http.StatusBadRequest)
        return
      }
      ErrorWithJSON(w, "Database Error", http.StatusInternalServerError)
      log.Println("Failed to insert book", err)
      return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Location", r.URL.Path+"/"+book.ISBN)
    w.WriteHeader(http.StatusCreated)
  }
}

func bookByISBN(s *mgo.Session) func(w http.ResponseWriter, r *http.Request){
  return func(w http.ResponseWriter, r *http.Request){
    session := s.Copy()
    defer session.Close()
    isbn := pat.Param(r, "isbn")
    c := session.DB("store").C("books")
    var book Book
    err := c.Find(bson.M{"isbn":isbn}).One(&book)
    if err != nil{
      ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
      log.Println("Failed to find book", err)
      return
    }
    if book.ISBN == ""{
      ErrorWithJSON(w, "Book not found", http.StatusNotFound)
      return
    }
    respBody, err := json.MarshalIndent(book, "", " ")
    if err != nil{
      log.Fatal(err)
    }
    ResponseWithJSON(w, respBody, http.StatusOK)
  }
}

func updateBook(s *mgo.Session) func(w http.ResponseWriter, r *http.Request){
  return func(w http.ResponseWriter, r *http.Request){
    session := s.Copy()
    defer session.Close()

    isbn := pat.Param(r, "isbn")
    var book Book
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&book)

    if err != nil {
			ErrorWithJSON(w, "Incorrect body", http.StatusBadRequest)
			return
		}
    c := session.DB("store").C("books")
    err = c.Update(bson.M{"isbn":isbn}, &book)
    if err != nil{
      switch err {
			default:
				ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
				log.Println("Failed update book: ", err)
				return
			case mgo.ErrNotFound:
				ErrorWithJSON(w, "Book not found", http.StatusNotFound)
				return
			}
    }
    w.WriteHeader(http.StatusNoContent)
  }
}

func deleteBook(s *mgo.Session) func(w http.ResponseWriter, r *http.Request){
  return func(w http.ResponseWriter, r *http.Request){
    session := s.Copy()
    defer session.Close()
    isbn := pat.Param(r, "isbn")
    c := session.DB("store").C("books")
    err := c.Remove(bson.M{"isbn":isbn})
    if err != nil {
      switch err {
			default:
				ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
				log.Println("Failed delete book: ", err)
				return
			case mgo.ErrNotFound:
				ErrorWithJSON(w, "Book not found", http.StatusNotFound)
				return
    }
  }
    w.WriteHeader(http.StatusNoContent)
  }
}



/*func bookByISBN(w http.ResponseWriter, r *http.Request){
  isbn := pat.Param(r, "isbn")
  for _, b := range bookstore{
    if b.ISBN == isbn {
      jsonOut, _ := json.Marshal(b)
      fmt.Fprintf(w, string(jsonOut))
      return
    }
  }
  w.WriteHeader(http.StatusNotFound)
}*/

/*func logging(h http.Handler) http.Handler{
  fn := func(w http.ResponseWriter, r *http.Request){
    fmt.Printf("Received request: %v\n", r.URL)
    h.ServeHTTP(w, r)
  }
  return http.HandlerFunc(fn)
}*/

/*
  session, err := mgo.Dial(mongoUrl)
  if err != nil {
    panic(err)
  }
  defer session.Close()

  session.SetMode(mgo.Monotonic, true)
  c := session.DB("test").C("people")
  err = c.Insert(&Person{"Ale", "+55 53 8116 9639"},
                  &Person{"Cla", "+55 53 8402 8510"})

  if err != nil{
    log.Fatal(err)
  }

  result := Person{}
  err = c.Find(bson.M{"Name":"Ale"}).One(&result)
  if err != nil{
    log.Fatal(err)
  }
  fmt.Println("Phone:", result.Phone)

}*/
/*
const (
	mongoUrl           = "localhost:27017"
	dbName             = "test_db"
	userCollectionName = "user"
)

func test() mgo.Index {

}*/
