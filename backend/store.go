package main

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v3"
)

var roomMembers xsync.MapOf[string, xsync.MapOf[uuid.UUID, int]]

// var userRooms xsync.MapOf[uuid.UUID, []string]: For WebSocket to consider

func CleanInactiveRoomsTask() {
	for {
		time.Sleep(10 * time.Minute)
		CleanInactiveRooms()
	}
}

func CleanInactiveRooms() {
	rows, err := findInactiveRoomsStmt.Query()
	if err != nil {
		log.Println("Failed to find inactive rooms!", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			log.Println("Failed to scan inactive room!", err)
			continue
		}

		if members, ok := roomMembers.Load(id); !ok || members.Size() == 0 {
			result, err := deleteRoomStmt.Exec(id)
			if err != nil {
				log.Println("Failed to delete inactive room!", err)
			} else if rows, err := result.RowsAffected(); err != nil || rows != 1 {
				log.Println("Failed to delete inactive room!", err)
			}
		}
	}
}