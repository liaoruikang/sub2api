import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AffiliateView from '../AffiliateView.vue'

const { copyToClipboard, createAffiliateWithdrawal, getAffiliateDetail, listAffiliateWithdrawals } = vi.hoisted(() => ({
  copyToClipboard: vi.fn(),
  createAffiliateWithdrawal: vi.fn(),
  getAffiliateDetail: vi.fn(),
  listAffiliateWithdrawals: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  default: {
    getAffiliateDetail,
    transferAffiliateQuota: vi.fn(),
    createAffiliateWithdrawal,
    listAffiliateWithdrawals,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    refreshUser: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('AffiliateView', () => {
  const affiliateCode = 'affiliate-code-that-is-long-enough-to-overflow-a-mobile-viewport'

  beforeEach(() => {
    vi.clearAllMocks()
    copyToClipboard.mockResolvedValue(true)
    getAffiliateDetail.mockResolvedValue({
      user_id: 1,
      aff_code: affiliateCode,
      inviter_id: null,
      aff_count: 0,
      aff_quota: 0,
      aff_frozen_quota: 0,
      aff_history_quota: 0,
      effective_rebate_rate_percent: 10,
      withdrawal_config: {
        enabled: true,
        min_amount: 10,
        fee_rate: 1,
      },
      invitees: [],
    })
  })

  it('stacks long values and copy controls on mobile while retaining desktop rows', async () => {
    const wrapper = mount(AffiliateView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
        },
      },
    })

    await flushPromises()

    const values = wrapper.findAll('code')
    expect(values).toHaveLength(2)
    for (const value of values) {
      expect(value.classes()).toEqual(expect.arrayContaining([
        'min-w-0',
        'break-all',
        'sm:flex-1',
        'sm:truncate',
      ]))
      expect(Array.from(value.element.parentElement?.classList ?? [])).toEqual(expect.arrayContaining([
        'flex-col',
        'items-stretch',
        'sm:flex-row',
        'sm:items-center',
      ]))
    }

    const copyButtons = wrapper.findAll('button').filter((button) =>
      ['affiliate.copyCode', 'affiliate.copyLink'].includes(button.text()),
    )
    expect(copyButtons).toHaveLength(2)
    for (const button of copyButtons) {
      expect(button.classes()).toEqual(expect.arrayContaining([
        'w-full',
        'sm:w-auto',
        'sm:shrink-0',
      ]))
    }

    await copyButtons[0].trigger('click')
    await copyButtons[1].trigger('click')
    await flushPromises()

    expect(copyToClipboard).toHaveBeenNthCalledWith(1, affiliateCode, 'affiliate.codeCopied')
    expect(copyToClipboard).toHaveBeenNthCalledWith(
      2,
      `${window.location.origin}/register?aff=${encodeURIComponent(affiliateCode)}`,
      'affiliate.linkCopied',
    )
  })

  it('submits the gross amount while displaying the percentage fee payout', async () => {
    getAffiliateDetail.mockResolvedValue({
      user_id: 1,
      aff_code: affiliateCode,
      inviter_id: null,
      aff_count: 0,
      aff_quota: 100,
      aff_frozen_quota: 0,
      aff_history_quota: 100,
      effective_rebate_rate_percent: 10,
      withdrawal_config: { enabled: true, min_amount: 10, fee_rate: 1 },
      invitees: [],
    })
    createAffiliateWithdrawal.mockResolvedValue({
      id: 1,
      request_no: 'AW001',
      amount: 100,
      fee_rate: 1,
      fee_amount: 1,
      payout_amount: 99,
      alipay_account_masked: 'buy****om',
      status: 'pending',
      created_at: '2026-08-16T00:00:00Z',
      updated_at: '2026-08-16T00:00:00Z',
    })

    const wrapper = mount(AffiliateView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
        },
      },
    })
    await flushPromises()

    await wrapper.get('#affiliate-withdrawal-amount').setValue('100')
    await wrapper.get('#affiliate-alipay-account').setValue('buyer@example.com')
    expect(wrapper.text()).toContain('$99.00')
    const submit = wrapper.findAll('button').find((button) => button.text() === 'affiliate.withdrawal.submit')
    expect(submit).toBeTruthy()
    await submit!.trigger('click')
    await flushPromises()

    expect(createAffiliateWithdrawal).toHaveBeenCalledWith({
      amount: 100,
      alipay_account: 'buyer@example.com',
    })
  })
})
