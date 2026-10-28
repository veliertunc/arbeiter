import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, View } from "react-native";
import { BrandHeader } from "../components/BrandHeader";
import { AuthCard } from "../components/AuthCard";
import { SecurityNote } from "../components/SecurityNote";
import { useAuth } from "../hooks/useAuth";

export function AuthScreen() {
  const auth = useAuth();
  return <KeyboardAvoidingView style={styles.fill} behavior={Platform.OS === "ios" ? "padding" : undefined}>
    <ScrollView contentContainerStyle={styles.page} keyboardShouldPersistTaps="handled">
      <BrandHeader />
      <View style={styles.hero}>
        <Text style={styles.eyebrow}>YOUR ACCOUNT, IN YOUR HANDS</Text>
        <Text style={styles.title}>{auth.user ? "You’re all set." : auth.isSignUp ? "Create your\naccount." : "Welcome\nback."}</Text>
        <Text style={styles.subtitle}>{auth.user ? "Your secure session is active on this device." : "Simple, secure access to your workspace."}</Text>
      </View>
      <AuthCard
        user={auth.user} email={auth.email} password={auth.password} name={auth.name} isSignUp={auth.isSignUp}
        busy={auth.busy} message={auth.message} setEmail={auth.setEmail} setPassword={auth.setPassword} setName={auth.setName}
        toggleMode={() => auth.setIsSignUp(!auth.isSignUp)} submit={auth.submit} signOut={auth.signOut}
      />
      <SecurityNote />
      <Text style={styles.footer}>ARBEITER AUTH · IDENTITY INFRASTRUCTURE</Text>
    </ScrollView>
  </KeyboardAvoidingView>;
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  page: { flexGrow: 1, paddingHorizontal: 24, paddingTop: 12, paddingBottom: 24, maxWidth: 520, width: "100%", alignSelf: "center" },
  hero: { paddingTop: 64, paddingBottom: 25 },
  eyebrow: { fontSize: 9, letterSpacing: 1.2, fontWeight: "700", color: "#6251d7" },
  title: { fontSize: 43, lineHeight: 48, fontWeight: "800", letterSpacing: -1.8, color: "#172133", marginTop: 13 },
  subtitle: { fontSize: 13, lineHeight: 20, color: "#7c8797", marginTop: 12 },
  footer: { fontSize: 8, color: "#a1a9b5", letterSpacing: 1, textAlign: "center", marginTop: "auto", paddingTop: 35 },
});
