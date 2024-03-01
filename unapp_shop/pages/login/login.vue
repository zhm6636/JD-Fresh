<template>
  <view class="container">
    <input v-model="mobile" placeholder="Username">
    <input v-model="password" placeholder="Password" type="password">
    <image :src="captchaUrl" @tap="refreshCaptcha"></image>
    <input v-model="captcha" placeholder="Captcha">
    <button @tap="login">Login</button>
  </view>
</template>

<script>
export default {
  data() {
    return {
      mobile: '',
      password: '',
      captcha: '',
      captchaUrl: '',
      captchaId: ''
    };
  },
  methods: {
    login() {
      // 在这里添加登录逻辑，例如调用后端API
      console.log('Logging in...');
      console.log(this.mobile, 1, this.password, 2, this.captcha, 3, this.captchaId, 4);
      uni.request({
        url: 'http://127.0.0.1:50001/u/v1/user/login',
        method: 'POST',
        data: {
          mobile: this.mobile,
          password: this.password,
          captcha: this.captcha,
          captchaId: this.captchaId
        },
        success: (res) => {

          if (res.data.code !== 200) {
            console.log(res.data.message)
            return
          }

          console.log('Login successful:', res);
          // 处理登录成功后的逻辑，例如，跳转到首页
          uni.setStorage({
            key: 'token',
            data: res.data.data,
            success: function () {
              console.log('success');
            }
          });
          uni.navigateTo({
            url: '/pages/index/index'
          });

        },
        fail: (err) => {
          console.error('Login failed:', err);
          // 处理登录失败的情况
        }
      });

      // 示例：模拟登录成功后刷新验证码
      this.refreshCaptcha();
    },
    refreshCaptcha() {
      // 调用后端生成图形验证码的API
      uni.request({
        url: 'http://127.0.0.1:50001/u/v1/base/captcha',
        method: 'GET',
        success: (res) => {
          // 更新captchaUrl，以刷新验证码图片
          console.log(res)
          this.captchaUrl = res.data.data;
          this.captchaId = res.data.id
        },
        fail: (err) => {
          console.error('Failed to get captcha:', err);
          // 处理获取验证码失败的情况
        }
      });
    }
  }
};
</script>

<style>
.container {
  padding: 20px;
}

input {
  margin-bottom: 10px;
}

button {
  background-color: #007aff;
  color: #fff;
  padding: 10px;
  border: none;
  border-radius: 5px;
}

image {
  width: 200px; /* 根据实际需求调整图片大小 */
  height: 80px;
}
</style>
