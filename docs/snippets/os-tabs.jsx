// Mintlify build warnings about Tab children can be safely ignored
export const OSTabs = ({ children, ...props }) => {
  function detectOS() {
    // Server-side safe OS detection
    if (typeof window === "undefined" || typeof navigator === "undefined") {
      return undefined; // Default to no selection
    }

    // If there's a hash in the URL, don't set a default (respect user's choice)
    if (window.location.hash) {
      return undefined;
    }

    const userAgent = window.navigator.userAgent;
    if (userAgent.includes("Mac")) {
      return 0; // macOS is first tab
    } else if (userAgent.includes("Win")) {
      return 1; // Windows is second tab
    }
    return 2; // Linux is third tab
  }

  const osIndex = detectOS();

  // Render Tabs with defaultTabIndex based on OS (or undefined if hash exists)
  return (
    <Tabs defaultTabIndex={osIndex} {...props}>
      {children}
    </Tabs>
  );
};
