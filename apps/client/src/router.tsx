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

const bookRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/book/$id',
  component: lazyRouteComponent(() => import('./routes/Book.js'), 'Book'),
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

const routeTree = rootRoute.addChildren([
  indexRoute,
  libraryRoute,
  seriesRoute,
  bookRoute,
  settingsRoute,
  collectionsRoute,
  collectionDetailRoute,
  readingListsRoute,
  readingListDetailRoute,
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
