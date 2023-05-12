package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = "3000"
	CONN_TYPE = "tcp"
)

type Exits struct {
	North int
	East  int
	South int
	West  int
}

type Room struct {
	Id                int
	Name              string
	Description       string
	ExtraDescriptions map[string]string
	Exits
}

type Player struct {
	Id   int
	Name string
	Conn net.Conn
	*Room
}

var players []*Player
var rooms []*Room

func buildTheWorld() {
	rooms = append(rooms, &Room{
		Id:          1,
		Name:        "An empty room",
		Description: "You are standing in a nice big empty room. There is nothing on the walls, nothing on the floor, and nothing on the ceiling. There is an open door to the west.",
		ExtraDescriptions: map[string]string{
			"walls":   "There is nothing on the walls.",
			"floor":   "There is nothing on the floor.",
			"ceiling": "There is nothing on the ceiling.",
			"door":    "You see a small dark room on the other side of the door.",
		},
		Exits: Exits{
			West: 2,
		},
	}, &Room{
		Id:          2,
		Name:        "A dark room",
		Description: "You are standing in a small, dark room. There is a glow coming from an open door to the east.",
		ExtraDescriptions: map[string]string{
			"door": "You see a large empty room on the other side of the door.",
		},
		Exits: Exits{
			East: 1,
		},
	})
}

func broadcast(player *Player, message string) {
	for _, recipient := range players {
		if recipient.Id == player.Id {
			continue
		} else {
			recipient.sendMessage(message)
		}
	}
}

func (player *Player) sendToRoom(message string) {
	for _, recipient := range players {
		if recipient.Id == player.Id || recipient.Room.Id != player.Room.Id {
			continue
		} else {
			recipient.sendMessage(message)
		}
	}
}

func (player *Player) sendMessage(message string) {
	player.Conn.Write([]byte("\r" + message + "\r\n> "))
}

func (player *Player) prompt(message string) (result string) {
	player.Conn.Write([]byte("\r" + message + " "))
	scanner := bufio.NewScanner(player.Conn)
	scanner.Scan()
	return scanner.Text()
}

func (player *Player) chat(message string) {
	player.sendMessage("You: " + message)
	broadcast(player, player.Name+": "+message)
}

func (player *Player) look(arg string) {
	if len(arg) == 0 {
		playerList := ""
		for _, other := range players {
			if other.Id != player.Id && other.Room.Id == player.Room.Id {
				playerList = playerList + other.Name + " is standing here.\r\n"
			}
		}
		player.sendMessage(player.Room.Name + "\r\n" + player.Room.Description + "\r\n" + playerList)
	} else {
		if len(player.Room.ExtraDescriptions[arg]) > 0 {
			player.sendMessage(player.Room.ExtraDescriptions[arg])
		} else {
			player.sendMessage("There is nothing by the name of '" + arg + "' to look at here.")
		}
	}
}

func (player *Player) move(direction string) {
	oldRoom := player.Room
	newRoom := player.Room

	switch direction {
	case "north":
		if player.Room.North > 0 {
			newRoom = getRoom(player.Room.North)
		}
	case "east":
		if player.Room.East > 0 {
			newRoom = getRoom(player.Room.East)
		}
	case "west":
		if player.Room.West > 0 {
			newRoom = getRoom(player.Room.West)
		}
	case "south":
		if player.Room.South > 0 {
			newRoom = getRoom(player.Room.South)
		}
	}

	if oldRoom != newRoom {
		player.sendToRoom(player.Name + " left to the " + direction + ".")
		player.Room = newRoom
		player.sendToRoom(player.Name + " has arrived.")
		player.look("")
	}
}

func getRoom(id int) *Room {
	for _, room := range rooms {
		if room.Id == id {
			return room
		}
	}

	return nil
}

func (player *Player) quit() {
	broadcast(player, player.Name+" has logged off.")

	connection := player.Conn
	newPlayers := []*Player{}
	for _, remaining := range players {
		if remaining.Id != player.Id {
			newPlayers = append(newPlayers, remaining)
		}
	}

	players = newPlayers
	connection.Close()
}

func (player *Player) welcome() {
	name := player.prompt("Hi! Please enter your name:")
	player.Name = name
	player.sendMessage("Welcome " + name + "! Type 'quit' to quit.\r\n\r\n")
}

func (player *Player) handleInput(input string) {
	command := strings.Fields(input)[0]
	args := strings.TrimPrefix(input, command+" ")
	switch command {
	case "quit":
		player.quit()
	case "chat":
		player.chat(args)
	case "look":
		player.look(args)
	case "north", "east", "south", "west":
		player.move(command)
	default:
		player.sendMessage("Sorry, " + command + " isn't a valid command.")
	}
}

func main() {
	// Build the world
	buildTheWorld()

	// Listen for incoming connections
	listener, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error listening: ", err.Error())
		os.Exit(1)
	}

	//Close listener when we close
	defer listener.Close()

	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)
	for {
		// Listen for new connections
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}

		newPlayer := Player{Id: len(players), Conn: conn, Room: rooms[0]}
		players = append(players, &newPlayer)

		go handleRequest(&newPlayer)
	}
}

func handleRequest(player *Player) {
	player.welcome()
	player.look("")

	scanner := bufio.NewScanner(player.Conn)
	for scanner.Scan() {
		player.Conn.Write([]byte("> "))
		input := scanner.Text()
		player.handleInput(input)
	}
}
