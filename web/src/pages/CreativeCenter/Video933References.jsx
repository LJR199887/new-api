import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  read933MediaDuration,
  validate933References,
} from '../../constants/video933';

// Remounted on model/mode changes so unfinished reads cannot attach to another model.
export default function Video933References({
  kind,
  items,
  onChange,
  upload,
  onBusy,
}) {
  const { t } = useTranslation();
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const [url, setURL] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const label = kind === 'audio' ? t('音频参考') : t('视频参考');
  const add = async (source) => {
    if (busy) return;
    setBusy(true);
    onBusy(true);
    setError('');
    try {
      if (typeof source === 'string' && !/^https?:\/\//i.test(source))
        throw new Error(t('请填写 HTTP(S) 素材链接'));
      if (typeof source !== 'string' && source.size > 200 * 1024 * 1024)
        throw new Error(t('素材大小不能超过 200MB'));
      const duration = await read933MediaDuration(source, kind);
      if (!mounted.current) return;
      const next = [...items, { url: '', duration }];
      validate933References(next);
      next[next.length - 1].url =
        typeof source === 'string' ? source : (await upload(source)).url;
      if (!mounted.current) return;
      onChange(next);
      setURL('');
    } catch (e) {
      if (mounted.current) setError(t(e.message));
    } finally {
      if (mounted.current) {
        setBusy(false);
        onBusy(false);
      }
    }
  };
  return (
    <section className='mt-3 rounded-xl border border-slate-200 p-3 text-xs'>
      <p className='mb-2 font-semibold'>
        {label} · {t('最多 3 个，单个 2–15 秒，同类总时长不超过 15 秒')}
      </p>
      <div className='flex gap-2'>
        <input
          aria-label={label}
          className='min-w-0 flex-1 rounded border px-2 py-1'
          placeholder='https://…'
          value={url}
          onChange={(e) => setURL(e.target.value)}
          disabled={busy}
        />
        <button
          type='button'
          disabled={busy || !url.trim() || items.length >= 3}
          onClick={() => add(url.trim())}
        >
          {t('添加')}
        </button>
        <label>
          {t('上传')}
          <input
            aria-label={t('上传') + label}
            type='file'
            accept={`${kind}/*`}
            disabled={busy || items.length >= 3}
            className='max-w-40'
            onChange={(e) => {
              const file = e.target.files?.[0];
              e.target.value = '';
              if (file) add(file);
            }}
          />
        </label>
      </div>
      {items.map((item, index) => (
        <div key={item.url + index} className='mt-2 flex gap-2'>
          <span className='min-w-0 flex-1 truncate'>
            {index + 1}. {item.url} ({item.duration.toFixed(2)}s)
          </span>
          <button
            type='button'
            disabled={busy}
            onClick={() => onChange(items.filter((_, i) => i !== index))}
          >
            {t('删除')}
          </button>
        </div>
      ))}
      {busy && <p role='status'>{t('正在读取或上传素材')}</p>}
      {error && (
        <p role='alert' className='mt-2 text-red-500'>
          {error}
        </p>
      )}
    </section>
  );
}
