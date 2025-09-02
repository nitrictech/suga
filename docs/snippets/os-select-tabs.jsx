// Note: Mintlify will show "Import source must start with '/snippets/'" warning
// but React hooks still work fine in this file.
import { useRef, useEffect } from "react";

export const OSSelectTabs = ({ children }) => {
  const containerRef = useRef(null);

  function detectOS() {
    const userAgent =
      typeof window !== "undefined" ? window.navigator.userAgent : "";
    if (userAgent.includes("Mac")) {
      return "macOS";
    } else if (userAgent.includes("Win")) {
      return "Windows";
    }
    return "Linux";
  }

  useEffect(() => {
    // Auto-select OS tab only if no hash is present (respects user navigation)
    if (!window.location.hash && containerRef.current) {
      const osLabel = detectOS();

      // Small delay to ensure tabs are rendered
      const timeoutId = setTimeout(() => {
        const allButtons = containerRef.current.querySelectorAll("li button");
        const tabButton = Array.from(allButtons).find(
          (btn) => btn.textContent.trim() === osLabel
        );
        if (tabButton) {
          tabButton.click();
        }
      }, 50); // timeout prevents scroll shift

      return () => clearTimeout(timeoutId);
    }
  }, []);

  return <div ref={containerRef}>{children}</div>;
};
