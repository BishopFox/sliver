import { Pipe, PipeTransform } from '@angular/core';


@Pipe({ name: 'fileSize' })
export class FileSizePipe implements PipeTransform {

  private units = [
    'bytes',
    'KB',
    'MB',
    'GB',
    'TB',
    'PB'
  ];

  transform(bytes: number = 0, precision: number = 2): string {
    if (isNaN(parseFloat(String(bytes))) || !isFinite(bytes)) {
      return '?';
    }
    let unit = 0;
    while (bytes >= 1024) {
      bytes /= 1024;
      unit++;
    }
    return bytes.toFixed(+ precision) + ' ' + this.units[unit];
  }
}

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
