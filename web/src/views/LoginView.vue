<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)

const form = reactive({
  user_name: '',
  password: '',
})

async function handleSubmit() {
  loading.value = true
  try {
    await auth.login(form)
    router.push('/tasks')
  } catch (err) {
    ElMessage.error((err as Error).message || '登入失敗')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <el-card class="login-card">
      <h2>排程管理系統登入</h2>
      <el-form :model="form" label-position="top" @submit.prevent="handleSubmit">
        <el-form-item label="使用者名稱">
          <el-input v-model="form.user_name" />
        </el-form-item>
        <el-form-item label="密碼">
          <el-input v-model="form.password" type="password" show-password @keyup.enter="handleSubmit" />
        </el-form-item>
        <el-button type="primary" :loading="loading" style="width: 100%" @click="handleSubmit">登入</el-button>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
}
.login-card {
  width: 360px;
}
</style>
