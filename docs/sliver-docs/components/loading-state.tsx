import { Spinner } from "@heroui/react";
import React from "react";

const LoadingState: React.FC = () => {
  return (
    <div className="flex min-h-screen w-full items-center justify-center">
      <Spinner label="Loading ..." labelColor="foreground" />
    </div>
  );
};

export default LoadingState;
