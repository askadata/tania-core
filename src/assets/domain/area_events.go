package domain

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type AreaCreated struct {
	UID          uuid.UUID
	Name         string
	Type         AreaType
	Location     AreaLocation
	Size         AreaSize
	FarmUID      uuid.UUID
	ReservoirUID uuid.UUID
	CreatedDate  time.Time
}

type AreaPhotoAdded struct {
	AreaUID  uuid.UUID
	Filename string
	MimeType string
	Size     int
	Width    int
	Height   int
}

type AreaNoteAdded struct {
	AreaUID      uuid.UUID
	UID          uuid.UUID
	Content      string
	CreatedDated time.Time
}

type AreaNoteRemoved struct {
	AreaUID uuid.UUID
	UID     uuid.UUID
}