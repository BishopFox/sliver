import { Spinner } from "@heroui/react";
import React from "react";

const LoadingState: React.FC = () => {
  return (
    <div className="flex min-h-[55vh] w-full items-center justify-center gap-3 text-muted">
      <Spinner aria-label="Loading" />
      <span className="text-sm">Loading documentation…</span>
    </div>
  );
};

export default LoadingState;
