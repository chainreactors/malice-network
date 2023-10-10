package encoders

import "github.com/gofrs/uuid"

func UUID() string {
	id, _ := uuid.NewV4()
	return id.String()
}
