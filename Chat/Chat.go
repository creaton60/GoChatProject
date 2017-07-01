package main

import (
	"container/list"
	"log"
	"net/http"
	"time"

	//"github.com/googollee/go-socket.io" //use socket.io
	"ChatPractice/Chat/src/github.com/googollee/go-socket.io"
)

//Define chat event struct
type Event struct {
	EvtType string //Event Type
	User string //User Name
	Timestamp int //Time Value
	Text string // Message
}

//Define subscriber
type Subscription struct {
	Archive []Event // Until now stack event storage use slice
	New <-chan Event //New Event Create -> Get Data
}

/**
 * This Function Generate New Event
 */
func NewEvent(evtType, user, msg string) Event{
	return Event{evtType, user, int(time.Now().Unix()), msg}
}

var (
	subscribe = make(chan (chan <- Subscription), 10) //Subscribe Channel
	unSubscribe = make(chan (<-chan Event), 10)  //UnSubscribe Channel
	publish = make(chan Event, 10) //Event Generate Channel
)

/**
 * This function working is add new user -> event subscribe
 */
func Subscribe() Subscription {
	c := make(chan Subscription)
	subscribe <- c
	return <- c
}

/*
 * This function working is withdraw user -> event unSubscribe
 */
func (s Subscription) Cancel(){
	unSubscribe <- s.New // Send to Channel unSubsribe

	for{   //무한루프
		select {
		case _, ok := <-s.New: //channel take all value
			if !ok {	// if take all data out function
				return
			}
		default:
			return
		}
	}
}

/**
 * This function working is join user -> event publish
 */
func Join(user string){
	publish <- NewEvent("join", user, "")
}

/**
 * This function working is user send chatting message  -> event publish
 */
func Say(user, message string){
	publish <- NewEvent("message", user,message)
}

/**
 * This function working is leave user  -> event publish
 */
func Leave(user string){
	publish <- NewEvent("leave", user,"")
}

/*
 * This function event process to sub , unsub , publish event
 */
func Chatroom(){
	archive := list.New()
	subscribers := list.New()

	for{
		select{
		case c := <-subscribe:
			var events []Event

			//stack event check
			for e := archive.Front(); e != nil; e = e.Next() {
				//save events slice
				events = append(events, e.Value.(Event))
			}

			subscriber := make(chan Event, 10) //Event Channel Create
			subscribers.PushBack(subscriber) //Event Channel add Subscriber list

			c <- Subscription{events, subscriber} //subscribe struct instance create and send channel
			case event := <-publish:
			for e := subscribers.Front(); e != nil; e = e.Next() {
				subscriber := e.Value.(chan Event)

				subscriber <- event
			}
				//save event length over 20
			if archive.Len() >= 20{
				archive.Remove(archive.Front()) //delete event
			}
			archive.PushBack(event) //current event save

			case c:= <-unSubscribe: //leave user
				for e:= subscribers.Front(); e != nil; e =e.Next(){
					subscriber := e.Value.(chan Event)

					//if sub list event equals channel c
					if subscriber == c {
						subscribers.Remove(e) //delete to sub list
						break
					}
				}

		}
	}
}

func main() {
	server, err := socketio.NewServer(nil) //socket io init

	if err != nil {
		log.Fatal(err)
	}

	go Chatroom()

	server.On("connection", func(so socketio.Socket){
		log.Println("Connection Server....")

		s := Subscribe() //subscribe process
		Join(so.Id())

		for _, event := range s.Archive {
			so.Emit("event", event)
		}

		newMessages := make(chan string)

		so.On("message", func(msg string){
			newMessages <- msg
		})

		so.On("disconnection", func(){
			Leave(so.Id())
			s.Cancel()
		})

		go func(){
			for{
				select {
				case event := <-s.New:
					so.Emit("event", event)
				case msg := <-newMessages:
					Say(so.Id(), msg)
				}
			}
		}()
	})

	http.Handle("/socekt.io/", server)

	http.Handle("/", http.FileServer(http.Dir(".")))

	http.ListenAndServe(":80", nil)
}