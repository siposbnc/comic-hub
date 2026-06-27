import { useEffect } from 'react';
import { SUPPORTED_EXTENSIONS } from '@comichub/reader-core';
import { useReaderStore } from './reader/store.js';
import { Reader } from './reader/Reader.js';
import { Button } from '@comichub/ui';
import { Icon } from '@comichub/ui';

export function App() {
  const status = useReaderStore((s) => s.status);
  const error = useReaderStore((s) => s.error);
  const init = useReaderStore((s) => s.init);
  const retry = useReaderStore((s) => s.retry);
  const dispose = useReaderStore((s) => s.dispose);

  useEffect(() => {
    void init();
    return () => dispose();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (status === 'ready') {
    return <Reader />;
  }

  if (status === 'loading') {
    return (
      <div className="screen" aria-busy="true">
        <span className="spinner spinner--lg" />
        <p className="screen__muted">Opening…</p>
      </div>
    );
  }

  if (status === 'error') {
    return (
      <div className="screen" role="alert">
        <Icon name="alert-triangle" size={40} />
        <h1 className="screen__title">Couldn&apos;t open this comic</h1>
        <p className="screen__muted">{error ?? 'Something went wrong.'}</p>
        <Button onClick={retry}>Try again</Button>
      </div>
    );
  }

  // idle / empty: launched without a file or a connected book.
  return (
    <div className="screen">
      <Icon name="book" size={44} />
      <h1 className="screen__title">ComicHub Reader</h1>
      <p className="screen__muted">
        Open a comic file ({SUPPORTED_EXTENSIONS.join(', ')}), or launch a book from the ComicHub
        client.
      </p>
      <p className="screen__hint">
        Dev: append <code>?bookId=&lt;id&gt;</code> (and optionally{' '}
        <code>&amp;server=&amp;token=&amp;page=</code>) to drive connected mode against a running
        server.
      </p>
    </div>
  );
}
