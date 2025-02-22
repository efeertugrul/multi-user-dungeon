package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	bolt "go.etcd.io/bbolt"
)

type Index struct {
	IndexID int64
	mu      sync.Mutex
}

func (i *Index) GetID() int64 {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.IndexID++
	return i.IndexID
}

func (i *Index) SetID(id int64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if id > i.IndexID {
		i.IndexID = id
	}
}

func (i *Index) Initialize(rooms map[int64]*Room) {

	var maxExitID int64

	for _, room := range rooms {
		for _, exit := range room.Exits {
			if exit.ExitID > maxExitID {
				maxExitID = exit.ExitID
			}
		}
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	i.IndexID = maxExitID + 1
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visible    bool
	Direction  string
}
type Room struct {
	RoomID      int64
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
}

func Display(rooms map[int64]*Room) {
	fmt.Println("Rooms:")
	for _, room := range rooms {
		fmt.Printf("Room %d: %s\n", room.RoomID, room.Title)
		for _, exit := range room.Exits {
			fmt.Printf("  Exit %s to room %d (%s)\n", exit.Direction, exit.TargetRoom, rooms[exit.TargetRoom].Title)
		}
	}
}

func LoadJSON(rooms map[int64]*Room, fileName string) (map[int64]*Room, error) {
	byteValue, err := os.ReadFile(fileName)
	if err != nil {
		return rooms, fmt.Errorf("error reading file: %w", err)
	}

	var data struct {
		Rooms map[string]struct {
			Area      string `json:"area"`
			Title     string `json:"title"`
			Narrative string `json:"description"`
			Exits     []struct {
				ExitName     string `json:"direction"`
				Visible      bool   `json:"visible"`
				TargetRoomID int64  `json:"target_room"`
			} `json:"exits"`
		} `json:"rooms"`
	}

	if err := json.Unmarshal(byteValue, &data); err != nil {
		return rooms, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	index := &Index{}
	index.Initialize(rooms)

	for id, roomData := range data.Rooms {
		roomID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return rooms, fmt.Errorf("error parsing room ID '%s': %w", id, err)
		}
		room := &Room{
			RoomID:      roomID,
			Area:        roomData.Area,
			Title:       roomData.Title,
			Description: roomData.Narrative,
			Exits:       make(map[string]*Exit),
		}

		rooms[roomID] = room

		for _, exitData := range roomData.Exits {
			exit := Exit{
				ExitID:     index.GetID(),
				TargetRoom: exitData.TargetRoomID,
				Visible:    exitData.Visible,
				Direction:  exitData.ExitName,
			}

			room.Exits[exit.Direction] = &exit
		}
	}

	return rooms, nil
}

func LoadBolt(rooms map[int64]*Room, fileName string) (map[int64]*Room, error) {
	db, err := bolt.Open(fileName, 0600, nil)
	if err != nil {
		fmt.Printf("Error opening BoltDB file: %v\n", err)
		return rooms, fmt.Errorf("error opening BoltDB file: %w", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			fmt.Println("Rooms bucket not found")
			return fmt.Errorf("Rooms bucket not found")
		}

		exitsBucket := tx.Bucket([]byte("Exits"))
		if exitsBucket == nil {
			fmt.Println("Exits bucket not found")
			return fmt.Errorf("Exits bucket not found")
		}

		err := roomsBucket.ForEach(func(k, v []byte) error {
			var room Room
			if err := json.Unmarshal(v, &room); err != nil {
				fmt.Printf("Error unmarshalling room data for key %s: %v\n", k, err)
				return fmt.Errorf("error unmarshalling room data: %w", err)
			}
			rooms[room.RoomID] = &room
			// fmt.Printf("Loaded Room %d: %+v\n", room.RoomID, room)
			return nil
		})
		if err != nil {
			return err
		}

		return exitsBucket.ForEach(func(k, v []byte) error {
			var exit Exit
			if err := json.Unmarshal(v, &exit); err != nil {
				fmt.Printf("Error unmarshalling exit data for key %s: %v\n", k, err)
				return fmt.Errorf("error unmarshalling exit data: %w", err)
			}

			keyParts := strings.SplitN(string(k), "_", 2)
			if len(keyParts) != 2 {
				fmt.Printf("Invalid exit key format: %s\n", k)
				return fmt.Errorf("invalid exit key format")
			}
			roomID, err := strconv.ParseInt(keyParts[0], 10, 64)
			if err != nil {
				fmt.Printf("Error parsing room ID from key %s: %v\n", k, err)
				return fmt.Errorf("error parsing room ID from key: %w", err)
			}

			if room, exists := rooms[roomID]; exists {
				room.Exits[exit.Direction] = &exit
				// fmt.Printf("Loaded Exit %s for Room %d: %+v\n", exit.Direction, room.RoomID, exit)
			} else {
				fmt.Printf("Room not found for exit key %s\n", k)
				return fmt.Errorf("room not found for exit: %s", string(k))
			}
			return nil
		})
	})

	if err != nil {
		fmt.Printf("Error reading from BoltDB: %v\n", err)
		return rooms, fmt.Errorf("error reading from BoltDB: %w", err)
	}

	// Display(rooms)

	return rooms, nil
}

func WriteBolt(rooms map[int64]*Room, dbPath string) (map[int64]*Room, error) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return rooms, err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		roomsBucket, err := tx.CreateBucketIfNotExists([]byte("Rooms"))
		if err != nil {
			fmt.Printf("Error creating 'Rooms' bucket: %v\n", err)
			return err
		}
		exitsBucket, err := tx.CreateBucketIfNotExists([]byte("Exits"))
		if err != nil {
			fmt.Printf("Error creating 'Exits' bucket: %v\n", err)
			return err
		}

		for _, room := range rooms {
			roomData, err := json.Marshal(room)
			if err != nil {
				fmt.Printf("Error marshalling room data (RoomID %d): %v\n", room.RoomID, err)
				return err
			}
			roomKey := strconv.FormatInt(room.RoomID, 10)
			err = roomsBucket.Put([]byte(roomKey), roomData)
			if err != nil {
				fmt.Printf("Error writing room data to 'Rooms' bucket: %v\n", err)
				return err
			}

			for _, exit := range room.Exits {
				exitData, err := json.Marshal(exit)
				if err != nil {
					fmt.Printf("Error marshalling exit data (ExitID %d): %v\n", exit.ExitID, err)
					return err
				}
				exitKey := fmt.Sprintf("%d_%s", room.RoomID, exit.Direction)
				err = exitsBucket.Put([]byte(exitKey), exitData)
				if err != nil {
					fmt.Printf("Error writing exit data to 'Exits' bucket: %v\n", err)
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return rooms, nil
}

func main() {
	jsonFilePath := flag.String("j", "test_data.json", "Path to the JSON file.")
	boltFilePath := flag.String("b", "test_data.bolt", "Path to the Bolt DB file.")
	help := flag.Bool("h", false, "Display help.")

	flag.Parse()

	if *help {
		fmt.Println("Usage of program:")
		fmt.Println("  -j string")
		fmt.Println("        Path to the JSON file. (default \"test_data.json\")")
		fmt.Println("  -b string")
		fmt.Println("        Path to the Bolt DB file. (default \"test_data.bolt\")")
		fmt.Println("  -h")
		fmt.Println("        Display help.")
		return
	}

	// Initialize the rooms map

	rooms := make(map[int64]*Room)

	// Load data from BoltDB
	rooms, err := LoadBolt(rooms, *boltFilePath)
	if err != nil {
		fmt.Println("Data load from BoltDB failed:", err)
	} else {
		fmt.Println("Data loaded from BoltDB successfully")
	}

	// Load the JSON data
	rooms, err = LoadJSON(rooms, *jsonFilePath)
	if err != nil {
		fmt.Println("Data load failed:", err)
	} else {
		fmt.Println("Data loaded successfully")
	}

	// Write data to BoltDB
	rooms, err = WriteBolt(rooms, *boltFilePath)
	if err != nil {
		fmt.Println("Data write failed:", err)
		return // Ensure to exit if writing fails
	} else {
		fmt.Println("Data written successfully")
	}

	// Load data from BoltDB
	rooms, err = LoadBolt(rooms, *boltFilePath)
	if err != nil {
		fmt.Println("Data load from BoltDB failed:", err)
	} else {
		fmt.Println("Data loaded from BoltDB successfully")
	}

}
