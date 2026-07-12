import {
  createRootRoute,
  createRoute,
  createRouter,
  lazyRouteComponent,
} from '@tanstack/react-router';
import { AppShell } from './components/AppShell.js';

const rootRoute = createRootRoute({ component: AppShell });

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: lazyRouteComponent(() => import('./routes/Home.js'), 'Home'),
});

const libraryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/library/$id',
  component: lazyRouteComponent(() => import('./routes/Library.js'), 'Library'),
});

const seriesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/series/$id',
  component: lazyRouteComponent(() => import('./routes/Series.js'), 'Series'),
});

const storyArcRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/series/$id/story-arcs/$arcId',
  component: lazyRouteComponent(() => import('./routes/Grouping.js'), 'StoryArc'),
});

const volumeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/series/$id/volumes/$volume',
  component: lazyRouteComponent(() => import('./routes/Grouping.js'), 'Volume'),
});

const bookRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/book/$id',
  component: lazyRouteComponent(() => import('./routes/Book.js'), 'Book'),
});

const trackerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/tracker',
  component: lazyRouteComponent(() => import('./routes/Tracker.js'), 'Tracker'),
});

const statsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/stats',
  component: lazyRouteComponent(() => import('./routes/Stats.js'), 'Stats'),
});

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  component: lazyRouteComponent(() => import('./routes/Settings.js'), 'Settings'),
});

const collectionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/collections',
  component: lazyRouteComponent(() => import('./routes/Collections.js'), 'Collections'),
});

const collectionDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/collections/$id',
  component: lazyRouteComponent(() => import('./routes/CollectionDetail.js'), 'CollectionDetail'),
});

const readingListsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/reading-lists',
  component: lazyRouteComponent(() => import('./routes/ReadingLists.js'), 'ReadingLists'),
});

const readingListDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/reading-lists/$id',
  component: lazyRouteComponent(() => import('./routes/ReadingListDetail.js'), 'ReadingListDetail'),
});

const smartListsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/smart-lists',
  component: lazyRouteComponent(() => import('./routes/SmartLists.js'), 'SmartLists'),
});

const smartListDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/smart-lists/$id',
  component: lazyRouteComponent(() => import('./routes/SmartListDetail.js'), 'SmartListDetail'),
});

const tagsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/tags',
  component: lazyRouteComponent(() => import('./routes/Tags.js'), 'Tags'),
});

const tagBooksRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/tags/$id',
  component: lazyRouteComponent(() => import('./routes/TagBooks.js'), 'TagBooks'),
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  libraryRoute,
  seriesRoute,
  storyArcRoute,
  volumeRoute,
  bookRoute,
  trackerRoute,
  statsRoute,
  settingsRoute,
  collectionsRoute,
  collectionDetailRoute,
  readingListsRoute,
  readingListDetailRoute,
  smartListsRoute,
  smartListDetailRoute,
  tagsRoute,
  tagBooksRoute,
]);

export const router = createRouter({
  routeTree,
  defaultPreload: 'intent',
  defaultPendingMinMs: 0,
});

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
