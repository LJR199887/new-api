/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useMemo } from 'react';
import { Button, Empty, Modal, Spin, Tag, Typography } from '@douyinfe/semi-ui';
import { Copy } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { copy, showError, showSuccess } from '../../../../helpers';

const TaskRequestModal = ({
  isRequestModalOpen,
  setIsRequestModalOpen,
  requestSnapshotLoading,
  requestSnapshot,
  requestTask,
}) => {
  const { t } = useTranslation();
  const formattedBody = useMemo(() => {
    if (!requestSnapshot?.recorded) {
      return '';
    }
    return JSON.stringify(requestSnapshot.body ?? {}, null, 2);
  }, [requestSnapshot]);

  const handleCopy = async () => {
    if (!formattedBody) {
      return;
    }
    if (await copy(formattedBody)) {
      showSuccess(t('复制成功'));
    } else {
      showError(t('复制失败，请手动复制'));
    }
  };

  const metadata = [
    [t('任务ID'), requestTask?.task_id || requestSnapshot?.task_id || '-'],
    [t('用户'), requestTask?.username || requestTask?.user_id || '-'],
    [
      t('模型'),
      requestTask?.properties?.origin_model_name ||
        requestTask?.properties?.upstream_model_name ||
        '-',
    ],
    [t('请求方法'), requestSnapshot?.method || '-'],
    [t('请求路径'), requestSnapshot?.request_path || '-'],
    [t('内容类型'), requestSnapshot?.content_type || '-'],
  ];

  return (
    <Modal
      title={t('任务请求详情')}
      visible={isRequestModalOpen}
      onCancel={() => setIsRequestModalOpen(false)}
      footer={null}
      width={760}
      bodyStyle={{ maxHeight: '76vh', overflow: 'auto', padding: 20 }}
    >
      {requestSnapshotLoading ? (
        <div style={{ minHeight: 260, display: 'grid', placeItems: 'center' }}>
          <Spin size='large' tip={t('加载中...')} />
        </div>
      ) : !requestSnapshot?.recorded ? (
        <Empty description={t('该任务未记录请求内容或快照已过期')} />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
              gap: '10px 20px',
            }}
          >
            {metadata.map(([label, value]) => (
              <div key={label} style={{ minWidth: 0 }}>
                <Typography.Text type='tertiary' size='small'>
                  {label}
                </Typography.Text>
                <Typography.Paragraph
                  ellipsis={{ rows: 2, showTooltip: true }}
                  style={{ margin: '2px 0 0' }}
                >
                  {String(value)}
                </Typography.Paragraph>
              </div>
            ))}
          </div>

          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 12,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <Typography.Text strong>{t('请求内容')}</Typography.Text>
              {requestSnapshot.truncated && (
                <Tag color='orange'>{t('已截断')}</Tag>
              )}
            </div>
            <Button
              theme='borderless'
              type='tertiary'
              icon={<Copy size={16} />}
              onClick={handleCopy}
            >
              {t('复制')}
            </Button>
          </div>

          <pre
            style={{
              margin: 0,
              padding: 16,
              maxHeight: '48vh',
              overflow: 'auto',
              border: '1px solid var(--semi-color-border)',
              borderRadius: 6,
              background: 'var(--semi-color-fill-0)',
              fontSize: 13,
              lineHeight: 1.55,
              whiteSpace: 'pre-wrap',
              overflowWrap: 'anywhere',
            }}
          >
            {formattedBody}
          </pre>
        </div>
      )}
    </Modal>
  );
};

export default TaskRequestModal;
