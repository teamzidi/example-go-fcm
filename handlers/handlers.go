//go:build !mock

package handlers

import (
	"github.com/teamzidi/example-go-fcm/fcm"
)

type fcmClient = *fcm.Client
