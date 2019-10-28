package log

/*
   Copyright 2019 Bruno Moura <brunotm@gmail.com>

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// Level represents the supported log levels
type Level uint8

const (
	// DEBUG log level
	DEBUG = Level(1)
	// INFO log level
	INFO = Level(2)
	// WARN log level
	WARN = Level(3)
	// ERROR log level
	ERROR = Level(4)

	maxLevel = int(ERROR)
)

func (l Level) String() (level string) {
	switch l {
	case DEBUG:
		return "debug"
	case INFO:
		return "info"
	case WARN:
		return "warn"
	case ERROR:
		return "error"
	default:
		return "unknow"
	}
}
