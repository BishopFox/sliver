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
