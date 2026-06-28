import { useState } from 'react';
import { Button, Input, Select, IconButton } from '@comichub/ui';
import type { SmartField, SmartOp, SmartRules, Tag } from '@comichub/api-client';

const FIELDS: { value: SmartField; label: string }[] = [
  { value: 'tag', label: 'Tag' },
  { value: 'series', label: 'Series' },
  { value: 'publisher', label: 'Publisher' },
  { value: 'format', label: 'Format' },
  { value: 'ageRating', label: 'Age rating' },
  { value: 'readStatus', label: 'Read status' },
];

const TEXT_OPS: { value: SmartOp; label: string }[] = [
  { value: 'is', label: 'is' },
  { value: 'isNot', label: 'is not' },
  { value: 'contains', label: 'contains' },
];
const ENUM_OPS: { value: SmartOp; label: string }[] = [
  { value: 'is', label: 'is' },
  { value: 'isNot', label: 'is not' },
];

const READ_STATUSES = [
  { value: 'unread', label: 'unread' },
  { value: 'in_progress', label: 'in progress' },
  { value: 'read', label: 'read' },
];

type DraftRule = { field: SmartField; op: SmartOp; value: string };

function opsFor(field: SmartField) {
  return field === 'tag' || field === 'readStatus' ? ENUM_OPS : TEXT_OPS;
}

function defaultValue(field: SmartField, tags: Tag[]): string {
  if (field === 'readStatus') return 'unread';
  if (field === 'tag') return tags[0]?.id ?? '';
  return '';
}

/**
 * A rule builder for creating a smart list: a name, a match mode (all/any), and one or
 * more field/operator/value rows. Calls onCreate with the assembled rule set.
 */
export function SmartListBuilder({
  tags,
  onCreate,
  creating,
}: {
  tags: Tag[];
  onCreate: (name: string, rules: SmartRules) => void;
  creating: boolean;
}) {
  const [name, setName] = useState('');
  const [match, setMatch] = useState<'all' | 'any'>('all');
  const [rules, setRules] = useState<DraftRule[]>([
    { field: 'readStatus', op: 'is', value: 'unread' },
  ]);

  const patchRule = (i: number, patch: Partial<DraftRule>) =>
    setRules((rs) => rs.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));

  const changeField = (i: number, field: SmartField) =>
    patchRule(i, { field, op: opsFor(field)[0]!.value, value: defaultValue(field, tags) });

  const addRule = () => setRules((rs) => [...rs, { field: 'series', op: 'is', value: '' }]);
  const removeRule = (i: number) => setRules((rs) => rs.filter((_, idx) => idx !== i));

  const valid =
    name.trim().length > 0 && rules.length > 0 && rules.every((r) => r.value.trim().length > 0);

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!valid || creating) return;
    onCreate(name.trim(), { match, rules });
    setName('');
    setRules([{ field: 'readStatus', op: 'is', value: 'unread' }]);
  };

  return (
    <form
      onSubmit={submit}
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 14,
        maxWidth: 720,
        marginBottom: 28,
        padding: 16,
        background: 'var(--surface-raised)',
        border: '1px solid var(--border-hairline)',
        borderRadius: 'var(--radius-lg)',
      }}
    >
      <div style={{ display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
        <div style={{ flex: 1, minWidth: 180 }}>
          <Input
            placeholder="Smart list name…"
            value={name}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
          />
        </div>
        <span style={{ fontSize: 'var(--text-small)', color: 'var(--text-tertiary)' }}>Match</span>
        <Select
          value={match}
          size="sm"
          onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
            setMatch(e.target.value as 'all' | 'any')
          }
        >
          <option value="all">all rules</option>
          <option value="any">any rule</option>
        </Select>
      </div>

      {rules.map((r, i) => (
        <div key={i} style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
          <Select
            value={r.field}
            size="sm"
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
              changeField(i, e.target.value as SmartField)
            }
          >
            {FIELDS.map((f) => (
              <option key={f.value} value={f.value}>
                {f.label}
              </option>
            ))}
          </Select>
          <Select
            value={r.op}
            size="sm"
            onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
              patchRule(i, { op: e.target.value as SmartOp })
            }
          >
            {opsFor(r.field).map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </Select>
          <div style={{ flex: 1, minWidth: 160 }}>
            <RuleValue rule={r} tags={tags} onChange={(value) => patchRule(i, { value })} />
          </div>
          <IconButton
            icon="x"
            label="Remove rule"
            variant="ghost"
            size="sm"
            onClick={() => removeRule(i)}
          />
        </div>
      ))}

      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 10 }}>
        <Button type="button" variant="ghost" icon="plus" onClick={addRule}>
          Add rule
        </Button>
        <Button type="submit" variant="secondary" disabled={!valid || creating}>
          {creating ? 'Creating…' : 'Create smart list'}
        </Button>
      </div>
    </form>
  );
}

function RuleValue({
  rule,
  tags,
  onChange,
}: {
  rule: DraftRule;
  tags: Tag[];
  onChange: (value: string) => void;
}) {
  if (rule.field === 'readStatus') {
    return (
      <Select
        value={rule.value}
        size="sm"
        onChange={(e: React.ChangeEvent<HTMLSelectElement>) => onChange(e.target.value)}
      >
        {READ_STATUSES.map((s) => (
          <option key={s.value} value={s.value}>
            {s.label}
          </option>
        ))}
      </Select>
    );
  }
  if (rule.field === 'tag') {
    if (tags.length === 0) {
      return (
        <span style={{ fontSize: 'var(--text-small)', color: 'var(--text-tertiary)' }}>
          No tags yet — add some from a book first.
        </span>
      );
    }
    return (
      <Select
        value={rule.value}
        size="sm"
        onChange={(e: React.ChangeEvent<HTMLSelectElement>) => onChange(e.target.value)}
      >
        {tags.map((t) => (
          <option key={t.id} value={t.id}>
            {t.name}
          </option>
        ))}
      </Select>
    );
  }
  return (
    <Input
      placeholder="value…"
      value={rule.value}
      onChange={(e: React.ChangeEvent<HTMLInputElement>) => onChange(e.target.value)}
    />
  );
}
