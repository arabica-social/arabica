import type { Preview } from "@storybook/sveltekit";

import { session, type Session } from "../src/lib/stores/session";

import "../../internal/web/assets/css/tokens.css";
import "../../internal/web/assets/css/reset.css";
import "../../internal/web/assets/css/utilities.css";

// Stories must render with the same cascade as the app, so the component
// layers are imported in the sorted order internal/web/assets/bundle.go
// concatenates them. bundle.go is the source of truth for the order.
import.meta.glob("../../internal/web/assets/css/components/*.css", {
  eager: true,
});

const signedOut: Session = {
  did: "",
  handle: "",
  displayName: "",
  avatar: "",
  isAuthenticated: false,
  isModerator: false,
  unreadNotifications: 0,
  temperatureUnit: "recorded",
};

const signedIn: Session = {
  did: "did:plc:storybook-viewer",
  handle: "viewer.arabica.example",
  displayName: "Pat",
  avatar: "",
  isAuthenticated: true,
  isModerator: false,
  unreadNotifications: 2,
  temperatureUnit: "recorded",
};

const preview: Preview = {
  parameters: {
    controls: { matchers: { color: /(background|color)$/i, date: /Date$/i } },
  },
  globalTypes: {
    theme: {
      name: "Theme",
      description: "Arabica light and dark themes",
      toolbar: {
        title: "Theme",
        icon: "circlehollow",
        items: ["light", "dark"],
        dynamicTitle: true,
      },
    },
    session: {
      name: "Session",
      description:
        "Body data-attributes the Go shell injects; components reading the session store react to this",
      toolbar: {
        title: "Session",
        icon: "user",
        items: [
          { value: "out", title: "Visitor" },
          { value: "in", title: "Signed in (demo)" },
        ],
        dynamicTitle: true,
      },
    },
  },
  initialGlobals: { theme: "light", session: "out" },
  decorators: [
    (Story, context) => {
      document.documentElement.setAttribute(
        "data-theme",
        context.globals.theme,
      );
      session.set(context.globals.session === "in" ? signedIn : signedOut);
      return Story();
    },
  ],
};

export default preview;
