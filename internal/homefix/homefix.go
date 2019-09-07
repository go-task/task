// Package homefix exists to address a bug where mvdan.cc/sh expects
// $HOME to be available in order to be able to expand "~".
//
// This should delete this package once this is fixed there.
package homefix
