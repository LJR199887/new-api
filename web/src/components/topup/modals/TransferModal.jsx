/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Modal, Typography, Input, InputNumber } from '@douyinfe/semi-ui';
import { CreditCard } from 'lucide-react';
import {
  quotaToDisplayAmount,
  displayAmountToQuota,
} from '../../../helpers/quota';
import { getCurrencyConfig } from '../../../helpers/render';

const TransferModal = ({
  t,
  openTransfer,
  transfer,
  handleTransferCancel,
  userState,
  renderQuota,
  getQuotaPerUnit,
  transferAmount,
  setTransferAmount,
}) => {
  const currencyConfig = getCurrencyConfig();
  const useTokenDisplay = currencyConfig.type === 'TOKENS';
  const minTransferQuota = getQuotaPerUnit();
  const maxTransferQuota = userState?.user?.aff_quota || 0;
  const transferDisplayAmount =
    transferAmount === '' || transferAmount == null
      ? ''
      : Number(quotaToDisplayAmount(transferAmount).toFixed(2));
  const minTransferDisplayAmount = Number(
    quotaToDisplayAmount(minTransferQuota).toFixed(2),
  );
  const maxTransferDisplayAmount = Number(
    quotaToDisplayAmount(maxTransferQuota).toFixed(2),
  );

  return (
    <Modal
      title={
        <div className='flex items-center'>
          <CreditCard className='mr-2' size={18} />
          {t('划转邀请额度')}
        </div>
      }
      visible={openTransfer}
      onOk={transfer}
      onCancel={handleTransferCancel}
      maskClosable={false}
      centered
    >
      <div className='space-y-4'>
        <div>
          <Typography.Text strong className='block mb-2'>
            {t('可用邀请额度')}
          </Typography.Text>
          <Input
            value={renderQuota(userState?.user?.aff_quota)}
            disabled
            className='!rounded-lg'
          />
        </div>
        <div>
          <Typography.Text strong className='block mb-2'>
            {t('划转额度')} · {t('最低') + renderQuota(getQuotaPerUnit())}
          </Typography.Text>
          <InputNumber
            min={minTransferDisplayAmount}
            max={maxTransferDisplayAmount}
            value={transferDisplayAmount}
            precision={useTokenDisplay ? 0 : 2}
            step={useTokenDisplay ? 1 : 0.01}
            prefix={useTokenDisplay ? undefined : currencyConfig.symbol}
            onChange={(value) =>
              setTransferAmount(
                value === '' || value == null ? '' : displayAmountToQuota(value),
              )
            }
            className='w-full !rounded-lg'
          />
        </div>
      </div>
    </Modal>
  );
};

export default TransferModal;
