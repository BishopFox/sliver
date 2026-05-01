import { Spinner } from "@heroui/react";
import React from "react";

const LoadingState: React.FC = () => {
  return (
    <div className="flex min-h-screen w-full items-center justify-center gap-3 text-foreground">
      <Spinner aria-label="Loading" />
      <span>Loading...</span>
    </div>
  );
};

export default LoadingState;
