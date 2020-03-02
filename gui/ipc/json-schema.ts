import * as Ajv from 'ajv';

// jsonSchema - A JSON Schema decorator, somewhat redundant given we're using TypeScript
// but it provides a stricter method of validating incoming JSON messages than simply
// casting the result of JSON.parse() to an interface.
export function jsonSchema(schema: object) {
    const ajv = new Ajv({allErrors: true});
    schema["additionalProperties"] = false;
    const validate = ajv.compile(schema);
    return (target: any, propertyKey: string, descriptor: PropertyDescriptor) => {
  
      const originalMethod = descriptor.value;
      descriptor.value = (arg: string) => {
        const valid = validate(arg);
        if (valid) {
          return originalMethod(arg);
        } else {
          console.error(validate.errors);
          return Promise.reject(`Invalid schema: ${ajv.errorsText(validate.errors)}`);
        }
      };
  
      return descriptor;
    };
  }
  