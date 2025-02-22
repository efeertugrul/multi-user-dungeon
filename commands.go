package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

var valid_commands []string = []string{"look", "go", "get", "drop", "inventory", "help", "quit", "say"}

// Function to check if a slice contains a specific string
func contains(slice []string, str string) bool {
	lowerStr := strings.ToLower(str)
	for _, v := range slice {
		if strings.ToLower(v) == lowerStr {
			return true
		}
	}
	return false
}

// Function to process the command
func validateCommand(command string, validCommands []string) (string, []string, error) {
	trimmedCommand := strings.TrimSpace(command)
	tokens := strings.Fields(trimmedCommand)

	if len(tokens) == 0 {
		return "", nil, errors.New("\n\rNo command entered.\n\r")
	}

	verb := ""

	// Convert all tokens to lowercase
	for i, token := range tokens {
		tokens[i] = strings.ToLower(token)
	}

	// Iterate through the tokens to find the first valid verb
	for _, token := range tokens {
		if contains(validCommands, token) {
			verb = token
			break
		}
	}

	if verb == "" {
		return verb, tokens, errors.New("\n\rI don't understand your command.")
	}

	return verb, tokens, nil
}

func executeCommand(character *Character, verb string, tokens []string) bool {

	command := strings.ToLower(verb)

	switch command {
	case "quit":
		return executeQuitCommand(character)

	case "say":
		return executeSayCommand(character, tokens)

	case "look":
		return executeLookCommand(character)

	case "help":
		return executeHelpCommand(character)

	default:
		character.Player.ToPlayer <- "\n\rCommand not yet implemented.\n\r"
	}

	return false // Indicate that the loop should continue
}

func executeQuitCommand(character *Character) bool {
	log.Printf("Player %s is quitting", character.Player.Name)
	character.Player.ToPlayer <- "\n\rGoodbye!"
	return true // Indicate that the loop should be exited
}

func executeSayCommand(character *Character, tokens []string) bool {
	if len(tokens) < 2 {
		character.Player.ToPlayer <- "\n\rWhat do you want to say?\n\r"
		return false
	}

	message := strings.Join(tokens[1:], " ")
	broadcastMessage := fmt.Sprintf("\n\r%s says: %s\n\r", character.Name, message)

	character.Player.Server.Mutex.Lock()
	for _, p := range character.Player.Server.Players {
		if p != character.Player {
			// Send message and prompt to other players
			p.ToPlayer <- broadcastMessage + p.Prompt
		}
	}
	character.Player.Server.Mutex.Unlock()

	// Send only the broadcast message to the player who issued the command
	character.Player.ToPlayer <- fmt.Sprintf("\n\rYou say: %s\n\r", message)

	return false
}

func executeLookCommand(character *Character) bool {
	room := character.Room
	character.Player.ToPlayer <- room.RoomInfo(character)
	return false
}

func executeHelpCommand(character *Character) bool {
	helpMessage := "\n\rAvailable Commands:" +
		"\n\rquit - Quit the game" +
		"\n\rsay <message> - Say something to all players" +
		"\n\rlook - Look around the room" +
		"\n\rhelp - Display available commands\n\r"

	character.Player.ToPlayer <- helpMessage
	return false
}
