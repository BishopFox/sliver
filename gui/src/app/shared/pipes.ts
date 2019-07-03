/*
  Sliver Implant Framework
  Copyright (C) 2019  Bishop Fox
  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.
  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.
  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import { Pipe, PipeTransform } from '@angular/core';

declare var moment: any;

/*
 * For nullable values will display a word instead of ''
 *
 * Usage:
 *   value | nullable:nullWord
*/

@Pipe({ name: 'nullable' })
export class NullablePipe implements PipeTransform {
  transform(value: string | null, nullWord: string): string {
    return value === null ? nullWord : value;
  }
}


@Pipe({ name: 'capitalize' })
export class CapitalizePipe implements PipeTransform {
  transform(value: string): string {
    return value[0].toUpperCase() + value.slice(1);
  }
}

@Pipe({ name: 'base64decode' })
export class Base64DecodePipe implements PipeTransform {
  transform(value: string): string {
    return atob(value);
  }
}

@Pipe({ name: 'unixtime' })
export class UnixTimePipe implements PipeTransform {
  transform(value: number | null, format = 'MMM Do YYYY h:mm a'): string {
    if (value === null) {
      return 'Never';
    } else if (typeof value === 'string') {
      return value;
    }
    return  moment(value * 1000).format(format);
  }
}
