/*
 * Copyright (C) 2026 Russ Shingleton <reshingleton@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package plugin

import (
	"encoding/json"
	"fmt"
)

// Bridge handles serialisation / deserialisation of data between the Go host
// and WASM guest modules. All data crossing the WASM boundary is encoded as
// compact JSON byte slices.
//
// The reason for a dedicated Bridge (rather than raw encoding/json calls) is
// to provide a single, consistent place to swap the wire format to Protobuf or
// MessagePack in the future without touching call sites.

// Bridge is the data-exchange layer between Go host and WASM guests.
type Bridge struct{}

// NewBridge returns a new Bridge instance.
func NewBridge() *Bridge {
	return &Bridge{}
}

// MarshalContentPayload serialises a ContentPayload to JSON bytes.
func (b *Bridge) MarshalContentPayload(p ContentPayload) ([]byte, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("bridge: marshal ContentPayload: %w", err)
	}
	return data, nil
}

// UnmarshalContentPayload deserialises JSON bytes into a ContentPayload.
func (b *Bridge) UnmarshalContentPayload(data []byte) (ContentPayload, error) {
	var p ContentPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("bridge: unmarshal ContentPayload: %w", err)
	}
	return p, nil
}

// MarshalAssetPayload serialises an AssetPayload to JSON bytes.
func (b *Bridge) MarshalAssetPayload(p AssetPayload) ([]byte, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("bridge: marshal AssetPayload: %w", err)
	}
	return data, nil
}

// UnmarshalAssetPayload deserialises JSON bytes into an AssetPayload.
func (b *Bridge) UnmarshalAssetPayload(data []byte) (AssetPayload, error) {
	var p AssetPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("bridge: unmarshal AssetPayload: %w", err)
	}
	return p, nil
}

// MarshalStoragePayload serialises a StoragePayload to JSON bytes.
func (b *Bridge) MarshalStoragePayload(p StoragePayload) ([]byte, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("bridge: marshal StoragePayload: %w", err)
	}
	return data, nil
}

// UnmarshalStoragePayload deserialises JSON bytes into a StoragePayload.
func (b *Bridge) UnmarshalStoragePayload(data []byte) (StoragePayload, error) {
	var p StoragePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("bridge: unmarshal StoragePayload: %w", err)
	}
	return p, nil
}

// MarshalError creates a JSON-encoded error response.
func (b *Bridge) MarshalError(err error) []byte {
	data, _ := json.Marshal(map[string]string{"error": err.Error()})
	return data
}

// IsErrorResponse returns true if the byte slice represents a bridge error
// response (i.e. has an "error" key at root level).
func (b *Bridge) IsErrorResponse(data []byte) (bool, string) {
	var v map[string]json.RawMessage
	if err := json.Unmarshal(data, &v); err != nil {
		return false, ""
	}
	if raw, ok := v["error"]; ok {
		var msg string
		_ = json.Unmarshal(raw, &msg)
		return true, msg
	}
	return false, ""
}
