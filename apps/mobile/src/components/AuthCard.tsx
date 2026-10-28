import { ActivityIndicator, StyleSheet, Text, TextInput, TouchableOpacity, View } from "react-native";
import type { User } from "../types";

type AuthCardProps = {
  user: User | null;
  email: string;
  password: string;
  name: string;
  isSignUp: boolean;
  busy: boolean;
  message: string;
  setEmail: (value: string) => void;
  setPassword: (value: string) => void;
  setName: (value: string) => void;
  toggleMode: () => void;
  submit: () => void;
  signOut: () => void;
};

export function AuthCard(props: AuthCardProps) {
  if (props.busy && !props.user) return <View style={styles.card}><ActivityIndicator color="#5b49dc" style={{ marginVertical: 35 }}/></View>;
  return <View style={styles.card}>
    {props.user ? <View style={styles.account}>
      <View style={styles.avatar}><Text style={styles.avatarText}>{props.user.name?.[0]?.toUpperCase() ?? "A"}</Text></View>
      <Text style={styles.accountName}>{props.user.name || "Your account"}</Text>
      <Text style={styles.accountEmail}>{props.user.email}</Text>
      <View style={styles.verified}><View style={styles.dot}/><Text style={styles.verifiedText}>{props.user.email_verified ? "Email verified" : "Session authenticated"}</Text></View>
      <TouchableOpacity style={styles.primary} onPress={() => void props.signOut()}><Text style={styles.primaryText}>Sign out</Text></TouchableOpacity>
    </View> : <>
      <Text style={styles.formTitle}>{props.isSignUp ? "Get started" : "Sign in"}</Text>
      <Text style={styles.formHint}>{props.isSignUp ? "Set up your Arbeiter account." : "Enter your details to continue."}</Text>
      {props.isSignUp && <><Text style={styles.label}>NAME <Text style={styles.optional}>OPTIONAL</Text></Text><TextInput style={styles.input} placeholder="Your name" value={props.name} onChangeText={props.setName} autoCapitalize="words"/></>}
      <Text style={styles.label}>EMAIL ADDRESS</Text>
      <TextInput style={styles.input} placeholder="you@example.com" value={props.email} onChangeText={props.setEmail} keyboardType="email-address" autoCapitalize="none" autoComplete="email"/>
      <Text style={styles.label}>PASSWORD</Text>
      <TextInput style={styles.input} placeholder={props.isSignUp ? "At least 12 characters" : "Your password"} value={props.password} onChangeText={props.setPassword} secureTextEntry autoComplete={props.isSignUp ? "new-password" : "current-password"}/>
      <TouchableOpacity style={styles.primary} onPress={() => void props.submit()} disabled={props.busy}>
        <Text style={styles.primaryText}>{props.busy ? "Connecting…" : props.isSignUp ? "Create account" : "Continue securely"}</Text><Text style={styles.arrow}>→</Text>
      </TouchableOpacity>
      <View style={styles.switchRow}><Text style={styles.switchText}>{props.isSignUp ? "Already have an account?" : "New to Arbeiter?"}</Text><Text onPress={props.toggleMode} style={styles.switchAction}>{props.isSignUp ? "Sign in" : "Create account"}</Text></View>
    </>}
    {props.message ? <Text style={styles.message}>{props.message}</Text> : null}
  </View>;
}

const styles = StyleSheet.create({
  card: { backgroundColor: "white", borderWidth: 1, borderColor: "#eceef4", borderRadius: 13, padding: 20, shadowColor: "#252e4b", shadowOpacity: .05, shadowRadius: 18, elevation: 2 },
  formTitle: { fontSize: 17, fontWeight: "700", color: "#1a2434" },
  formHint: { fontSize: 10, color: "#8b95a4", marginTop: 4, marginBottom: 20 },
  label: { fontSize: 8, letterSpacing: .8, color: "#647083", fontWeight: "700", marginBottom: 6, marginTop: 12 },
  optional: { color: "#a1a9b5", fontWeight: "500" },
  input: { height: 43, borderWidth: 1, borderColor: "#e7e9f0", borderRadius: 7, paddingHorizontal: 12, fontSize: 12, color: "#243044", backgroundColor: "#fff" },
  primary: { height: 45, backgroundColor: "#5b49dc", borderRadius: 7, marginTop: 20, alignItems: "center", justifyContent: "center", flexDirection: "row" },
  primaryText: { color: "white", fontSize: 11, fontWeight: "700" },
  arrow: { color: "white", position: "absolute", right: 15 },
  switchRow: { flexDirection: "row", justifyContent: "center", gap: 5, marginTop: 17 },
  switchText: { fontSize: 9, color: "#8d97a6" },
  switchAction: { fontSize: 9, color: "#5b49dc", fontWeight: "700" },
  account: { alignItems: "center", paddingVertical: 10 },
  avatar: { width: 55, height: 55, borderRadius: 28, backgroundColor: "#e9e6ff", alignItems: "center", justifyContent: "center" },
  avatarText: { fontSize: 21, color: "#5946d6", fontWeight: "700" },
  accountName: { fontSize: 15, fontWeight: "700", color: "#1a2434", marginTop: 12 },
  accountEmail: { fontSize: 10, color: "#8490a0", marginTop: 4 },
  verified: { flexDirection: "row", alignItems: "center", gap: 6, paddingVertical: 9 },
  verifiedText: { fontSize: 9, color: "#568a73" },
  dot: { width: 6, height: 6, borderRadius: 4, backgroundColor: "#25b982" },
  message: { fontSize: 10, color: "#5b49dc", textAlign: "center", marginTop: 13 },
});
