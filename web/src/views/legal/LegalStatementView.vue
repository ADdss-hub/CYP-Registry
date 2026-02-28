<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import CypButton from '@/components/common/CypButton.vue'
import { LEGAL_STATEMENT_STORAGE_KEY, LEGAL_STATEMENT_VERSION } from '@/constants/legal'

const router = useRouter()

const isAcknowledged = ref(false)

function checkAcknowledged() {
  isAcknowledged.value = localStorage.getItem(LEGAL_STATEMENT_STORAGE_KEY) === '1'
}

function handleAgree() {
  localStorage.setItem(LEGAL_STATEMENT_STORAGE_KEY, '1')
  isAcknowledged.value = true
  router.push('/')
}

onMounted(() => {
  checkAcknowledged()
  if (isAcknowledged.value) {
    router.replace('/')
  }
})
</script>

<template>
  <div class="legal-page">
    <div class="legal-container">
      <header class="page-header">
        <h1 class="page-title">个人声明与数据处理规范</h1>
        <p class="page-subtitle">
          请在继续使用 CYP-Registry 前，仔细阅读并确认以下声明内容（当前版本：{{ LEGAL_STATEMENT_VERSION }}）。
        </p>
      </header>

      <section class="statement-section">
        <div class="statement-card">
          <h2>一、用户行为规范（节选）</h2>
          <p>
            用户在使用 CYP-Registry（以下简称“本产品”）时，必须遵守《中华人民共和国网络安全法》《网络信息内容生态治理规定》等法律法规及本规范，不得利用本产品从事任何违法违规行为。
          </p>
          <p>
            包括但不限于：发布、传播违法、暴力、色情、低俗、谣言、诽谤、侵权类信息；窃取、篡改、破坏他人数据或本产品功能；实施网络攻击、诈骗、洗钱等活动；规避安全机制、破解或篡改产品代码等。
          </p>
        </div>

        <div class="statement-card">
          <h2>二、免责声明（节选）</h2>
          <p>
            本产品提供的内容、功能及服务，均基于个人开发与公开信息整理优化，虽尽力保障信息准确完整，但不对其时效性、适用性、无错误性、无侵权性作出任何明示或默示保证。
          </p>
          <p>
            用户因依赖本产品提供的信息、功能作出决策而产生的直接或间接损失，由用户自行承担；本人不对因网络不稳定、设备故障、版本兼容、操作失误、第三方工具干扰等引发的损失承担责任。
          </p>
          <p>
            本人作为个人开发者，不具备企业级安全防护能力，不承诺7×24小时运维保障及数据备份服务，用户需充分认识并接受相应使用风险。
          </p>
        </div>

        <div class="statement-card">
          <h2>三、数据处理规范（节选）</h2>
          <p>
            本人严格遵循“合法、正当、必要、诚信”原则处理用户个人信息，仅收集实现产品功能必需的信息，不超范围收集、使用、存储数据，全程以个人名义开展处理活动，无团队参与或数据共享行为。
          </p>
          <p>
            数据分类包括一般个人信息（如用户名、邮箱、设备型号等）和敏感个人信息（如身份证号、银行卡号、生物识别信息、位置轨迹等），敏感信息将采取高强度加密、单独授权、最短必要存储周期等专项保护措施。
          </p>
          <p>
            用户享有知情权、决定权、查询与复制权、更正与补充权、删除权、撤回同意权、账户注销权及数据可携带权，可通过文档中列明的联系方式行使上述权利。
          </p>
        </div>

        <div class="statement-card">
          <h2>四、其他说明</h2>
          <p>
            本界面仅展示经适配后的核心条款节选，完整文本及适配说明请参见项目根目录《个人全平台通用声明与数据处理模板》文档；如后续版本更新，将通过本界面或系统设置页进行显著提示。
          </p>
          <p>
            继续使用本产品即视为您已完整阅读并理解上述声明与数据处理规则，并同意在相关法律法规及本规范约束下使用本产品。
          </p>
        </div>

        <div class="statement-actions">
          <CypButton type="primary" size="large" @click="handleAgree">
            我已阅读并同意，进入控制台
          </CypButton>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped lang="scss">
.legal-page {
  display: flex;
  justify-content: center;
  padding: 24px;
}

.legal-container {
  width: 100%;
  max-width: 960px;
}

.page-header {
  margin-bottom: 24px;

  .page-title {
    font-size: 28px;
    font-weight: 700;
    color: #1e293b;
    margin: 0 0 8px;
  }

  .page-subtitle {
    font-size: 14px;
    color: #64748b;
    margin: 0;
  }
}

.statement-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.statement-card {
  background: #ffffff;
  border-radius: 12px;
  padding: 20px 24px;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);

  h2 {
    font-size: 18px;
    font-weight: 600;
    color: #1e293b;
    margin: 0 0 12px;
  }

  p {
    font-size: 14px;
    line-height: 1.6;
    color: #4b5563;
    margin: 0 0 8px;

    &:last-child {
      margin-bottom: 0;
    }
  }
}

.statement-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

@media (max-width: 768px) {
  .legal-page {
    padding: 16px;
  }

  .statement-card {
    padding: 16px 16px;
  }
}
</style>

