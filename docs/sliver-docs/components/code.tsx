import { Themes } from "@/util/themes";
import { faLinux, faPython } from "@fortawesome/free-brands-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import Editor, { loader } from "@monaco-editor/react";
import { Card, CardBody } from "@nextui-org/react";
import { useTheme } from "next-themes";
import React, { useRef } from "react";

export type CodeSchema = {
  name: string;
  script_type: string;
  source_code: string;
};

export type CodeViewerProps = {
  script: CodeSchema;

  hideHeader?: boolean;

  className?: string;
};

export function renderScriptTypeIcon(key: string, className?: string) {
  switch (key) {
    case "bash":
      return (
        <FontAwesomeIcon icon={faLinux} className={className || undefined} />
      );

    case "python":
      return (
        <FontAwesomeIcon icon={faPython} className={className || undefined} />
      );
    default:
      return <></>;
  }
}

const CodeViewer = (props: CodeViewerProps) => {
  const { theme } = useTheme();

  // Editor
  loader.config({ paths: { vs: "/js/monaco" } });
  const editorRef = useRef(null as any);
  function handleEditorDidMount(editor: any, monaco: any) {
    editorRef.current = editor;
  }
  const vsTheme = React.useMemo(() => {
    return theme === Themes.DARK ? "vs-dark" : undefined;
  }, [theme]);
  const language = React.useMemo(() => {
    switch (props.script?.script_type) {
      case "bash":
        return "shell";
      default:
        return props.script?.script_type || "shell";
    }
  }, [props]);

  const [scriptSourceCode, setScriptSourceCode] = React.useState(
    props.script.source_code
  );
  const [fontSize, setFontSize] = React.useState(14);
  const editorContainerClassName = React.useMemo(() => {
    return theme === Themes.DARK
      ? "col-span-12 mt-4 rounded-2xl overflow-hidden"
      : "col-span-12 mt-4 rounded-2xl overflow-hidden border border-gray-300";
  }, [theme]);

  return (
    <div className="grid grid-cols-12">
      {!props.hideHeader ? (
        <div className="col-span-12 mt-2">
          <Card>
            <CardBody>
              <div className="flex w-full items-center">
                <div className="w-full">
                  <div>
                    <div className="flex items-center text-xl monospace">
                      {renderScriptTypeIcon(
                        props.script.script_type || "",
                        "mr-2"
                      )}
                      {`${props.script.name}`}
                    </div>
                    <div className="flex w-full">
                      <div className="text-xs text-gray-500 capitalize">
                        {`${props.script.script_type} script - Version`}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </CardBody>
          </Card>
        </div>
      ) : (
        <></>
      )}
      <div className={editorContainerClassName}>
        <Editor
          className={props.className || "min-h-[200px]"}
          theme={vsTheme}
          defaultLanguage={language}
          defaultValue={scriptSourceCode}
          onChange={(value, event) => {
            setScriptSourceCode(value || "");
          }}
          onMount={handleEditorDidMount}
          options={{
            readOnly: true,
            fontFamily: "Fira Code",
            fontLigatures: true,
            fontSize: fontSize,
          }}
        />
      </div>
    </div>
  );
};

export default CodeViewer;
